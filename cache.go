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

const CACHESZ uint64 = 10

type Cache struct {
	mu      *sync.RWMutex
	entries []entry
}

func mkCache() *Cache {
	entries := make([]entry, CACHESZ)
	n := uint64(len(entries))
	for i := uint64(0); i < n; i++ {
		entries[i].cobj.mu = new(sync.RWMutex)
	}
	return &Cache{
		mu:      new(sync.RWMutex),
		entries: entries,
	}
}

// Conditionally allocate a cache slot for id
func (c *Cache) getputObj(id uint64) *Cobj {
	var hit *entry
	var empty *entry

	c.mu.Lock()
	n := uint64(len(c.entries))
	for i := uint64(0); i < n; i++ {
		if c.entries[i].id == id {
			hit = &c.entries[i]
			break
		}
		if c.entries[i].ref == 0 && !c.entries[i].pin && empty == nil {
			empty = &c.entries[i]
		}
		continue
	}
	if hit != nil {
		hit.ref = hit.ref + 1
		c.mu.Unlock()
		return &hit.cobj
	}
	if empty == nil {
		panic("getputObj")
		c.mu.Unlock()
		return nil
	}
	hit = empty
	hit.id = id
	hit.ref = 1
	hit.cobj.valid = false
	c.mu.Unlock()
	return &hit.cobj
}

// Lookup cache slot for id
func (c *Cache) getObj(id uint64) *Cobj {
	var hit *entry

	c.mu.Lock()
	n := uint64(len(c.entries))
	for i := uint64(0); i < n; i++ {
		if c.entries[i].ref > 0 && c.entries[i].id == id {
			hit = &c.entries[i]
			break
		}
		continue
	}
	if hit != nil {
		hit.ref = hit.ref + 1
		c.mu.Unlock()
		return &hit.cobj
	}
	c.mu.Unlock()
	return nil
}

// Decrease ref count of the cache slot for id
func (c *Cache) putObj(id uint64, pin bool) bool {
	var hit *entry
	var last bool

	c.mu.Lock()
	n := uint64(len(c.entries))
	for i := uint64(0); i < n; i++ {
		if c.entries[i].ref > 0 && c.entries[i].id == id {
			hit = &c.entries[i]
			break
		}
		continue
	}
	if hit != nil {
		hit.pin = pin
		hit.ref = hit.ref - 1
		if hit.ref == 0 {
			last = true
		} else {
			last = false
		}
	}
	if hit == nil {
		panic("putObj")
	}
	c.mu.Unlock()
	return last
}
