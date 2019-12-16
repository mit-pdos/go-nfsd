package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

type Addr struct {
	blkno uint64
	off   uint64 // offset in bits
	sz    uint64 // sz in bits
}

func (a *Addr) Eq(b Addr) bool {
	return a.blkno == b.blkno && a.off == b.off && a.sz == b.sz
}

func (a *Addr) Inc(start uint64, len uint64) {
	a.off = a.off + 1
	if a.off >= disk.BlockSize {
		a.off = 0
		a.blkno = a.blkno + 1
	}
	if a.blkno >= start+len {
		a.blkno = start
	}
}

func mkAddr(blkno uint64, off uint64, sz uint64) Addr {
	return Addr{blkno: blkno, off: off, sz: sz}
}

type AddrMap struct {
	mu   *sync.RWMutex
	bufs map[uint64][]*Buf
}

func mkAddrMap() *AddrMap {
	a := &AddrMap{
		mu:   new(sync.RWMutex),
		bufs: make(map[uint64][]*Buf),
	}
	return a
}

func (amap *AddrMap) Len() uint64 {
	amap.mu.Lock()
	l := uint64(len(amap.bufs))
	amap.mu.Unlock()
	return l
}

func (amap *AddrMap) LookupInternal(addr Addr) *Buf {
	var buf *Buf
	bs, ok := amap.bufs[addr.blkno]
	if ok {
		for _, b := range bs {
			if addr.Eq(b.addr) {
				buf = b
				break
			}
		}
	}
	return buf
}

func (amap *AddrMap) AddInternal(buf *Buf) {
	blkno := buf.addr.blkno
	amap.bufs[blkno] = append(amap.bufs[blkno], buf)
}

func (amap *AddrMap) Lookup(addr Addr) *Buf {
	amap.mu.Lock()
	buf := amap.LookupInternal(addr)
	amap.mu.Unlock()
	return buf
}

func (amap *AddrMap) LookupAdd(addr Addr, buf *Buf) bool {
	amap.mu.Lock()
	b := amap.LookupInternal(addr)
	if b == nil {
		amap.AddInternal(buf)
		amap.mu.Unlock()
		return true
	}
	DPrintf(5, "LookupAdd already locked %v %v\n", addr, b)
	amap.mu.Unlock()
	return false
}

func (amap *AddrMap) LookupBufs(blkno uint64) []*Buf {
	amap.mu.Lock()
	bs, _ := amap.bufs[blkno]
	amap.mu.Unlock()
	return bs
}

func (amap *AddrMap) Add(buf *Buf) {
	amap.mu.Lock()
	amap.AddInternal(buf)
	amap.mu.Unlock()
}

func (amap *AddrMap) Del(addr Addr) {
	var index uint64
	var found bool

	amap.mu.Lock()
	blkno := addr.blkno
	bs, found := amap.bufs[blkno]
	if !found {
		panic("Del")
	}
	for i, b := range bs {
		if b.addr.Eq(addr) {
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

func (amap *AddrMap) Dirty() uint64 {
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
