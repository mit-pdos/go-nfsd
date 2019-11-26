package goose_nfs

import (
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
	pin  bool   // is the slot pinned?
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

// Evict a not-inuse slot. Assume locked cache.
func (c *Cache) evict() {
	var addr uint64 = 0
	for a, entry := range c.entries {
		if entry.ref == 0 && !entry.pin {
			addr = a
			break
		}
		continue
	}
	if addr == 0 {
		panic("evict")
	}
	delete(c.entries, addr)
	c.cnt = c.cnt - 1
}

// Lookup the cache slot for id.  Create the slot if id isn't in the
// cache, perhaps evicting a not in-use slot in the cache.
func (c *Cache) lookupSlot(id uint64) *Cslot {
	c.mu.Lock()
	e := c.entries[id]
	if e != nil {
		e.ref = e.ref + 1
		c.mu.Unlock()
		return &e.slot
	}

	if c.cnt >= c.sz {
		c.evict()
	}
	s := Cslot{mu: new(sync.RWMutex), obj: nil}
	enew := &entry{ref: 1, pin: false, slot: s}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.slot
}

// Decrease ref count of the cache slot for id and entry.obj
// maybe deleted by evict
func (c *Cache) freeSlot(id uint64, pin bool) {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref - 1
		entry.pin = pin
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
	return false
}
