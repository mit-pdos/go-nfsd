package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

// Allocator keeps bitmap block in memory.
type Alloc struct {
	lock  *sync.RWMutex // protects head
	start uint64
	len   uint64
	head  Addr
}

func mkAlloc(start uint64, len uint64) *Alloc {
	head := mkAddr(start, 0, 1) // 1 byte
	a := &Alloc{
		lock:  new(sync.RWMutex),
		start: start,
		len:   len,
		head:  head,
	}
	return a
}

func findFreeBit(byteVal byte) (uint64, bool) {
	var off uint64
	var ok bool
	for bit := byte(0); bit < 8; bit++ {
		if byteVal&(byte(1)<<bit) == 0 {
			off = uint64(bit)
			ok = true
			break
		}
		continue
	}
	return off, ok
}

// Find a free bit in blk and toggle it
func findAndMark(blk disk.Block) (uint64, bool) {
	var off uint64
	var ok bool
	l := uint64(len(blk))
	for byte := uint64(0); byte < l; byte++ {
		byteVal := blk[byte]
		if byteVal == 0xff {
			continue
		}
		bit, bitOk := findFreeBit(byteVal)
		if bitOk {
			off = 8*byte + bit
			ok = true
			markBit(blk, off)
			break
		}
		// unreachable (since byte is not 0xff)
		continue
	}
	return off, ok
}

// Free bit bn in buf
func freeBit(buf *Buf, bn uint64) {
	if bn != buf.addr.off {
		panic("freeBit1")
	}
	bit := bn % 8
	buf.blk[0] = buf.blk[0] & ^(1 << bit)
}

// Alloc bit bn in blk
func markBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] = blk[byte] | (1 << bit)
}

// Find a region in the bitmap with some free bits. Assume caller
// holds allocator lock
func (a *Alloc) FindRegion(txn *Txn) *Buf {
	var buf *Buf
	var addr Addr
	a.lock.Lock()
	addr = a.head
	head := a.head
	for {
		buf = txn.ReadBuf(addr)
		a.head.Inc(a.start, a.len)
		if buf.blk[0] != byte(0xFF) {
			break
		}
		txn.RemBuf(buf)
		buf = nil
		addr = a.head
		if addr == head {
			break
		}
		continue
	}
	a.lock.Unlock()
	return buf
}

// Find a free region in the bitmap that is not locked by a
// transaction and lock it.
func (a *Alloc) LockFreeRegion(txn *Txn) *Buf {
	var buf *Buf
	for {
		buf = a.FindRegion(txn)
		if buf == nil {
			break
		}
		ok := txn.locked.LookupAdd(buf.addr, buf)
		if ok { // not locked?
			break
		}
		txn.RemBuf(buf)
		continue
	}
	log.Printf("LockFreeRegion: %v\n", buf)
	return buf
}

// Lock the region in the bitmap that contains n
func (a *Alloc) LockRegion(txn *Txn, n uint64) *Buf {
	var buf *Buf
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	buf = txn.ReadBufLocked(addr)
	log.Printf("LockRegion: %v\n", buf)
	return buf
}

func (a *Alloc) UnlockRegion(txn *Txn, buf *Buf) {
	log.Printf("UnlockRegion: %v\n", buf)
	txn.locked.Del(buf)
}

func (a *Alloc) Alloc(buf *Buf) uint64 {
	var n uint64 = 0

	b, found := findAndMark(buf.blk)
	if !found {
		return n
	}
	n = (buf.addr.blkno-a.start)*NBITBLOCK + buf.addr.off*8 + b
	return n
}

func (a *Alloc) Free(buf *Buf, n uint64) {
	i := n / NBITBLOCK
	log.Printf("Free1 buf %v %d %d\n", buf, n, i)
	if i >= a.len {
		panic("freeBlock")
	}
	if buf.addr.blkno != a.start+i {
		panic("freeBlock")
	}
	freeBit(buf, (n%NBITBLOCK)/8)
}

func (a *Alloc) RegionAddr(n uint64) Addr {
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	return addr
}

// XXX maybe a transaction thing
func (a *Alloc) AllocMyNum(txn *Txn, blkno uint64) uint64 {
	var n uint64 = 0
	bs := txn.amap.LookupBufs(blkno)
	for _, b := range bs {
		n = a.Alloc(b)
		if n != 0 {
			break
		}
	}
	return n
}

func (a *Alloc) AllocNum(txn *Txn) uint64 {
	var n uint64 = 0
	for i := a.start; i < a.start+a.len; i++ {
		n = a.AllocMyNum(txn, i)
		if n != 0 {
			break
		}

	}
	if n == 0 {
		b := a.LockFreeRegion(txn)
		if b != nil {
			n = a.Alloc(b)
			if n == 0 {
				panic("AllocInum")
			}
			b.Dirty()
		}
	}
	return n
}

func (a *Alloc) FreeNum(txn *Txn, num uint64) {
	if num == 0 {
		panic("FreeNum")
	}
	buf := a.LockRegion(txn, num)
	a.Free(buf, num)
	buf.Dirty()
}
