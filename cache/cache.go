package cache

import (
	"container/list"
	"sync"

	"github.com/mit-pdos/goose-nfsd/util"
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
	mu  *sync.Mutex // mutex protecting obj in this slot
	Obj interface{}
}

func (slot *Cslot) Lock() {
	slot.mu.Lock()
}

func (slot *Cslot) Unlock() {
	slot.mu.Unlock()
}

type entry struct {
	ref  uint32 // the slot's reference count
	slot Cslot
	lru  *list.Element
	id   uint64
}

type Cache struct {
	mu      *sync.Mutex
	entries map[uint64]*entry
	lru     *list.List
	sz      uint64
	cnt     uint64
}

func MkCache(sz uint64) *Cache {
	entries := make(map[uint64]*entry, sz)
	return &Cache{
		mu:      new(sync.Mutex),
		entries: entries,
		lru:     list.New(),
		cnt:     0,
		sz:      sz,
	}
}

func (c *Cache) PrintCache() {
	for k, v := range c.entries {
		util.DPrintf(0, "Entry %v %v\n", k, v.ref)
	}
}

func (c *Cache) evict() bool {
	e := c.lru.Front()
	if e == nil {
		return false
	}
	entry := e.Value.(*entry)
	c.lru.Remove(e)
	util.DPrintf(5, "evict: %d\n", entry.id)
	delete(c.entries, entry.id)
	c.cnt = c.cnt - 1
	return true
}

// Lookup the cache slot for id.  Create the slot if id isn't in the
// cache and if there is space in the cache. If no space, return
// nil to indicate the caller to evict entries.
func (c *Cache) LookupSlot(id uint64) *Cslot {
	c.mu.Lock()
	e := c.entries[id]
	if e != nil {
		if id != e.id {
			panic("LookupSlot")
		}
		if e.ref == 0 {
			c.lru.Remove(e.lru)
		}
		e.ref = e.ref + 1
		c.mu.Unlock()
		return &e.slot
	}
	if c.cnt >= c.sz {
		if !c.evict() {
			// failed to find victim. caller is
			// responsible for creating space.
			c.PrintCache()
			c.mu.Unlock()
			return nil
		}
	}
	enew := &entry{
		ref:  1,
		slot: Cslot{mu: new(sync.Mutex), Obj: nil},
		lru:  nil,
		id:   id,
	}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.slot
}

// Decrease ref count of the cache slot for id so that entry.obj maybe
// deleted by evict
func (c *Cache) FreeSlot(id uint64) {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref - 1
		util.DPrintf(5, "freeslot %d %d\n", id, entry.ref)
		if entry.ref == 0 {
			entry.lru = c.lru.PushBack(entry)
		}
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	panic("FreeSlot")
}
