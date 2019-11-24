package goose_nfs

import (
	"sync"
)

type Cobj struct {
	mu    *sync.RWMutex
	valid bool
	obj   interface{}
}

type entry struct {
	// cache info:
	id  uint64
	ref uint32
	pin bool
	// the entry
	cobj Cobj
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

// Assume locked cache
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

// Conditionally allocate a cache slot for id
func (c *Cache) getputObj(id uint64) *Cobj {
	c.mu.Lock()
	e := c.entries[id]
	if e != nil {
		e.ref = e.ref + 1
		c.mu.Unlock()
		return &e.cobj
	}

	if c.cnt >= c.sz {
		c.evict()
	}
	o := Cobj{mu: new(sync.RWMutex), valid: false, obj: nil}
	enew := &entry{id: id, ref: 1, pin: false, cobj: o}
	c.entries[id] = enew
	c.cnt = c.cnt + 1
	c.mu.Unlock()
	return &enew.cobj
}

// Lookup cache slot for id
func (c *Cache) getObj(id uint64) *Cobj {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref + 1
		c.mu.Unlock()
		return &entry.cobj
	}
	c.mu.Unlock()
	return nil
}

// Decrease ref count of the cache slot for id
func (c *Cache) putObj(id uint64, pin bool) bool {
	c.mu.Lock()
	entry := c.entries[id]
	if entry != nil {
		entry.ref = entry.ref - 1
		last := entry.ref == 0
		entry.pin = pin
		c.mu.Unlock()
		return last
	}
	c.mu.Unlock()
	panic("putObj")
	return false
}
