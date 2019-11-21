package goose_nfs

import (
	"log"
	"sync"
)

type Cobj struct {
	mu    *sync.RWMutex
	valid bool
	obj   interface{}
}

type entry struct {
	// cache info:
	mu  *sync.RWMutex
	id  uint64
	ref uint32
	// the entry
	cobj Cobj
}

const CACHESZ uint64 = 10

type Cache struct {
	lock    *sync.RWMutex
	entries []entry
}

func mkCache() *Cache {
	entries := make([]entry, CACHESZ)
	n := uint64(len(entries))
	for i := uint64(0); i < n; i++ {
		entries[i].mu = new(sync.RWMutex)
	}
	for i := uint64(0); i < n; i++ {
		entries[i].cobj.mu = new(sync.RWMutex)
	}
	return &Cache{
		lock:    new(sync.RWMutex),
		entries: entries,
	}
}

func (c *Cache) getObj(id uint64) *Cobj {
	var hit *entry

	c.lock.Lock()
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
		c.lock.Unlock()
		return &hit.cobj
	}
	c.lock.Unlock()
	return nil
}

func (c *Cache) getputObj(id uint64) *Cobj {
	var hit *entry
	var empty *entry

	c.lock.Lock()
	n := uint64(len(c.entries))
	for i := uint64(0); i < n; i++ {
		if c.entries[i].ref > 0 && c.entries[i].id == id {
			hit = &c.entries[i]
			break
		}
		if c.entries[i].ref == 0 && empty == nil {
			empty = &c.entries[i]
		}
		continue
	}
	if hit != nil {
		c.lock.Unlock()
		return &hit.cobj
	}
	if empty == nil {
		c.lock.Unlock()
		return nil
	}
	log.Printf("getput %d\n", id)
	hit = empty
	hit.id = id
	hit.ref = 1
	hit.cobj.valid = false
	c.lock.Unlock()
	return &hit.cobj
}
