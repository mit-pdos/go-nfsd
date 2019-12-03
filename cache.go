package goose_nfs

import (
	"log"
	"sync"
)

// A shared, fixed-size cache mapping from uint64 to a
// reference-counted slot in the cache.  The cache has a fixed number
// of slots.  A lookup for key, increments the reference count for
// that slot. Callers are responsible for filling a slot.  When a
// caller doesn't need the slot anymore (because it is done with the
// object in the slot), then caller must decrement the reference count
// for the slot.  When a reference counter for slot is 0, the cache
// can evict that slot, if it needs space for other objects.

type Cslot struct {
	mu  *sync.RWMutex // mutex protecting obj in this slot
	obj interface{}
}

func (slot *Cslot) lock() {
	slot.mu.Lock()
}

func (slot *Cslot) unlock() {
	slot.mu.Unlock()
}

type entry struct {
	ref  uint32 // the slot's reference count
	pin  TxnNum // is the slot pinned until transaction pin?
	slot Cslot
}

type Cache struct {
	mu      *sync.RWMutex
	entries map[uint64]*entry
	sz      uint64
	cnt     uint64
}

func mkCache(sz uint64) *Cache {
	entries := make(map[uint64]*entry, sz)
	return &Cache{
		mu:      new(sync.RWMutex),
		entries: entries,
		cnt:     0,
		sz:      sz,
	}
}

func (c *Cache) printCache() {
	for k, v := range c.entries {
		log.Printf("Entry %v: %v\n", k, v)
	}
}

func (c *Cache) evict() uint64 {
	var addr uint64 = 0
	for a, entry := range c.entries {
		if entry.ref == 0 && entry.pin == 0 {
			addr = a
			break
		}
		continue
	}
	if addr != 0 {
		log.Printf("findVictim: evict %d\n", addr)
		delete(c.entries, addr)
		c.cnt = c.cnt - 1
	}
	return addr
}

// Lookup the cache slot for id.  Create the slot if id isn't in the
// cache and if there is space in the cache. If no space, return
// nil to indicate the caller to evict entries.
func (c *Cache) lookupSlot(id uint64) *Cslot {
	c.mu.Lock()
	e := c.entries[id]
	if e != nil {
		e.ref = e.ref + 1
		c.mu.Unlock()
		return &e.slot
	}
	if c.cnt >= c.sz {
		if c.evict() == 0 {
			// failed to find victim. caller is
			// responsible for creating space.
			c.mu.Unlock()
			return nil
		}
	}
	s := Cslot{mu: new(sync.RWMutex), obj: nil}
	enew := &entry{ref: 1, pin: 0, slot: s}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.slot
}

// Decrease ref count of the cache slot for id so that entry.obj maybe
// deleted by evict
func (c *Cache) freeSlot(id uint64) {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref - 1
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	panic("putObj")
}

// Decrease ref count of the cache slot for id and return slot if
// last, so that caller can delete cached object. The caller should
// hold the locked obj in the slot.
func (c *Cache) delSlot(id uint64) bool {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref - 1
		last := entry.ref == 0
		c.mu.Unlock()
		return last
	}
	c.mu.Unlock()
	panic("delSlot")
}

// Pin ids belonging to txn
func (c *Cache) Pin(ids []uint64, txn TxnNum) {
	c.mu.Lock()
	log.Printf("Pin %d %v\n", txn, ids)
	for _, id := range ids {
		e := c.entries[id]
		e.pin = txn
	}
	c.mu.Unlock()
}

// Unpin ids through txn
func (c *Cache) UnPin(ids []uint64, txn TxnNum) {
	c.mu.Lock()
	log.Printf("Unpin through %d %v\n", txn, ids)
	for _, id := range ids {
		entry := c.entries[id]
		if entry == nil {
			log.Printf("Unpin %d isn't present\n", id)
			panic("Unpin")
		}
		if txn >= entry.pin {
			entry.pin = 0
		} else {
			log.Printf("Unpin: keep %d pinned at %d\n", id, entry.pin)
		}
	}
	c.mu.Unlock()
}
