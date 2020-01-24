package cache

import (
	"testing"
)

func TestCache(t *testing.T) {
	c := MkCache(10)
	for i := uint64(0); i < uint64(10); i++ {
		c.LookupSlot(i)
		c.FreeSlot(i)
	}
	if c.lru.Len() != 10 {
		t.Errorf("lru wrong")
	}
	c.LookupSlot(0)
	if c.lru.Len() != 9 {
		t.Errorf("lru too short")
	}
	c.FreeSlot(0)
	e := c.lru.Front()
	v := e.Value.(*entry)
	if v.id != 1 {
		t.Errorf("lru wrong head")
	}
	if !c.evict() {
		t.Errorf("evict failed")
	}
	if c.lru.Len() != 9 {
		t.Errorf("lru too short")
	}
	e = c.lru.Front()
	v = e.Value.(*entry)
	if v.id != 2 {
		t.Errorf("lru wrong head 2")
	}
}
