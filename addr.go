package goose_nfs

import (
	"sync"
)

type addr struct {
	blkno uint64
	off   uint64 // offset in bits
	sz    uint64 // sz in bits
}

func (a *addr) eq(b addr) bool {
	return a.blkno == b.blkno && a.off == b.off && a.sz == b.sz
}

func mkaddr(blkno uint64, off uint64, sz uint64) addr {
	return addr{blkno: blkno, off: off, sz: sz}
}

type addrMap struct {
	mu   *sync.RWMutex
	bufs map[uint64][]*buf
}

func mkaddrMap() *addrMap {
	a := &addrMap{
		mu:   new(sync.RWMutex),
		bufs: make(map[uint64][]*buf),
	}
	return a
}

func (amap *addrMap) len() uint64 {
	amap.mu.Lock()
	l := uint64(len(amap.bufs))
	amap.mu.Unlock()
	return l
}

func (amap *addrMap) lookupInternal(addr addr) *buf {
	var buf *buf
	bs, ok := amap.bufs[addr.blkno]
	if ok {
		for _, b := range bs {
			if addr.eq(b.addr) {
				buf = b
				break
			}
		}
	}
	return buf
}

func (amap *addrMap) addInternal(buf *buf) {
	blkno := buf.addr.blkno
	amap.bufs[blkno] = append(amap.bufs[blkno], buf)
}

func (amap *addrMap) lookup(addr addr) *buf {
	amap.mu.Lock()
	buf := amap.lookupInternal(addr)
	amap.mu.Unlock()
	return buf
}

func (amap *addrMap) lookupAdd(addr addr, buf *buf) bool {
	amap.mu.Lock()
	b := amap.lookupInternal(addr)
	if b == nil {
		amap.addInternal(buf)
		amap.mu.Unlock()
		return true
	} else {
		dPrintf(5, "LookupAdd already locked %v %v\n", addr, b)
	}
	dPrintf(5, "LookupAdd already locked %v %v\n", addr, b)
	amap.mu.Unlock()
	return false
}

func (amap *addrMap) add(buf *buf) {
	amap.mu.Lock()
	amap.addInternal(buf)
	amap.mu.Unlock()
}

func (amap *addrMap) del(addr addr) {
	var index uint64
	var found bool

	amap.mu.Lock()
	blkno := addr.blkno
	bs, found := amap.bufs[blkno]
	if !found {
		panic("Del")
	}
	for i, b := range bs {
		if b.addr.eq(addr) {
			index = uint64(i)
			found = true
		}
	}
	if !found {
		panic("Del")
	}
	bufs := append(bs[0:index], bs[index+1:]...)
	amap.bufs[blkno] = bufs
	amap.mu.Unlock()
}

func (amap *addrMap) dirty() uint64 {
	var n uint64 = 0
	amap.mu.Lock()
	for _, bs := range amap.bufs {
		for _, b := range bs {
			if b.dirty {
				n = n + 1
			}
		}
	}
	amap.mu.Unlock()
	return n
}
