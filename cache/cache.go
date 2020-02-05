package cache

import (
	"container/list"
	"sync"

	"github.com/mit-pdos/goose-nfsd/util"
)

type Cslot struct {
	Obj interface{}
}

type entry struct {
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
		util.DPrintf(0, "Entry %v %v\n", k, v)
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
		if e.lru != nil { // only remove on first lookupSlot
			c.lru.Remove(e.lru)
		}
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
		slot: Cslot{Obj: nil},
		lru:  nil,
		id:   id,
	}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.slot
}

// Done with a cache entry; put it back on lru list
func (c *Cache) Done(id uint64) {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		util.DPrintf(5, "freeslot %d %d\n", id)
		entry.lru = c.lru.PushBack(entry)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	panic("FreeSlot")
}
