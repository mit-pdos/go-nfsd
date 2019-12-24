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

type cslot struct {
	mu  *sync.Mutex // mutex protecting obj in this slot
	obj interface{}
}

func (slot *cslot) lock() {
	slot.mu.Lock()
}

func (slot *cslot) unlock() {
	slot.mu.Unlock()
}

type entry struct {
	ref  uint32 // the slot's reference count
	pin  txnNum // is the slot pinned until transaction pin?
	slot cslot
}

type cache struct {
	mu      *sync.Mutex
	entries map[uint64]*entry
	sz      uint64
	cnt     uint64
}

func mkCache(sz uint64) *cache {
	entries := make(map[uint64]*entry, sz)
	return &cache{
		mu:      new(sync.Mutex),
		entries: entries,
		cnt:     0,
		sz:      sz,
	}
}

func (c *cache) printCache() {
	for k, v := range c.entries {
		dPrintf(0, "Entry %v: %v\n", k, v)
	}
}

func (c *cache) evict() uint64 {
	var addr uint64 = 0
	var done = false
	for a, entry := range c.entries {
		if !done && entry.ref == 0 && entry.pin == 0 {
			addr = a
			done = true
		}
	}
	if addr != 0 {
		dPrintf(5, "evict: %d\n", addr)
		delete(c.entries, addr)
		c.cnt = c.cnt - 1
	}
	return addr
}

// Lookup the cache slot for id.  Create the slot if id isn't in the
// cache and if there is space in the cache. If no space, return
// nil to indicate the caller to evict entries.
func (c *cache) lookupSlot(id uint64) *cslot {
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
	s := cslot{mu: new(sync.Mutex), obj: nil}
	enew := &entry{ref: 1, pin: 0, slot: s}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.slot
}

// Decrease ref count of the cache slot for id so that entry.obj maybe
// deleted by evict
func (c *cache) freeSlot(id uint64) {
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

// Pin ids until txn < nexttxn have committed
func (c *cache) pin(ids []uint64, nxttxn txnNum) {
	c.mu.Lock()
	dPrintf(5, "Pin till nxttxn %d %v\n", nxttxn, ids)
	for _, id := range ids {
		e := c.entries[id]
		if e == nil {
			panic("Pin")
		}
		e.pin = nxttxn
	}
	c.mu.Unlock()
}

// Unpin ids through nxttxn
func (c *cache) unPin(ids []uint64, nxttxn txnNum) {
	c.mu.Lock()
	dPrintf(5, "Unpin through %d %v\n", nxttxn, ids)
	for _, id := range ids {
		entry := c.entries[id]
		if entry == nil {
			panic("Unpin")
		}
		if nxttxn >= entry.pin {
			entry.pin = 0
		} else {
			dPrintf(10, "Unpin: keep %d pinned at %d\n", id, entry.pin)
		}
	}
	c.mu.Unlock()
}
