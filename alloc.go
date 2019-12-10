package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

// Allocator keeps bitmap block in memory.  Allocator allocates
// tentatively in bmap and commits allocations to bmapCommit/bmap on
// commit. bmapCommit reflects the bmap state in commit order. Abort
// undoes changes to bmap.  Allocator delays freeing until commit, and
// then updates bmap.  XXX a lock per bit
type Alloc struct {
	lock       *sync.RWMutex
	bmap       []disk.Block
	bmapCommit []disk.Block
	start      uint64
	len        uint64
	head       Addr
	locked     *AddrMap
}

func mkAlloc(start uint64, len uint64) *Alloc {
	bmap := make([]disk.Block, len)
	bmapCommit := make([]disk.Block, len)
	for i := uint64(0); i < len; i++ {
		blkno := start + i
		bmap[i] = disk.Read(blkno)
		bmapCommit[i] = disk.Read(blkno)
	}
	head := mkAddr(start, 0, 1) // 1 byte
	a := &Alloc{
		lock:       new(sync.RWMutex),
		bmap:       bmap,
		bmapCommit: bmapCommit,
		start:      start,
		len:        len,
		head:       head,
		locked:     mkAddrMap(),
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

// Free bit bn in blk
func freeBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] = blk[byte] & ^(1 << bit)
}

// Free bit bn in buf
func freeBit1(buf *Buf, bn uint64) {
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

func (a *Alloc) markBlock(bn uint64) {
	i := bn / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}
}

// Assume caller holds allocator lock
func (a *Alloc) FindRegion(txn *Txn) *Buf {
	var buf *Buf
	var addr Addr
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
	return buf
}

func (a *Alloc) LockFreeRegion(txn *Txn) *Buf {
	var buf *Buf
	a.lock.Lock()
	for {
		buf = a.FindRegion(txn)
		if buf == nil {
			break
		}
		b := a.locked.Lookup(buf.addr)
		if b == nil { // not locked?
			break
		}
		txn.RemBuf(buf) // XXX panics
		continue
	}
	if buf != nil {
		a.locked.Add(buf)
	}
	a.lock.Unlock()
	log.Printf("LockFreeRegion: %v\n", buf)
	return buf
}

func (a *Alloc) LockRegion(txn *Txn, n uint64) *Buf {
	a.lock.Lock()
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	b := a.locked.Lookup(addr)
	if b != nil {
		panic("AllocRegion")
	}
	buf := txn.ReadBuf(addr)
	log.Printf("LockRegion: %v\n", buf)
	a.locked.Add(buf)
	a.lock.Unlock()
	return buf
}

func (a *Alloc) UnlockRegion(buf *Buf) {
	a.lock.Lock()
	log.Printf("UnlockRegion: %v\n", buf)
	a.locked.Del(buf)
	a.lock.Unlock()
}

// Zero indicates failure
func (a *Alloc) Alloc() uint64 {
	var bit uint64 = 0

	a.lock.Lock()
	for i := uint64(0); i < a.len; i++ {
		b, found := findAndMark(a.bmap[i])
		if !found {
			continue
		}
		bit = i*NBITBLOCK + b
		break
	}
	a.lock.Unlock()
	return bit
}

func (a *Alloc) Alloc1(buf *Buf) uint64 {
	var n uint64 = 0

	b, found := findAndMark(buf.blk)
	if !found {
		return n
	}
	n = (buf.addr.blkno-a.start)*NBITBLOCK + buf.addr.off*8 + b
	return n
}

func (a *Alloc) Free(n uint64) {
	a.lock.Lock()
	i := n / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}
	freeBit(a.bmap[i], n%NBITBLOCK)
	a.lock.Unlock()
}

func (a *Alloc) Free1(buf *Buf, n uint64) {
	i := n / NBITBLOCK
	log.Printf("Free1 buf %v %d %d\n", buf, n, i)
	if i >= a.len {
		panic("freeBlock")
	}
	if buf.addr.blkno != a.start+i {
		panic("freeBlock")
	}
	freeBit1(buf, (n%NBITBLOCK)/8)
}

func (a *Alloc) RegionAddr(n uint64) Addr {
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	return addr
}

func (a *Alloc) CommitBmap(alloc []uint64, free []uint64) []*Buf {
	bufs := make([]*Buf, 0)
	dirty := make([]bool, a.len)
	a.lock.Lock()
	for _, bn := range alloc {
		i := bn / NBITBLOCK
		dirty[i] = true
		markBit(a.bmapCommit[i], bn%NBITBLOCK)
	}
	for _, bn := range free {
		i := bn / NBITBLOCK
		dirty[i] = true
		freeBit(a.bmap[i], bn%NBITBLOCK)
		freeBit(a.bmapCommit[i], bn%NBITBLOCK)
	}
	for i, v := range dirty {
		if v {
			addr := mkAddr(a.start+uint64(i), 0, disk.BlockSize)
			buf := mkBuf(addr, a.bmapCommit[i], nil)
			bufs = append(bufs, buf)
		}
	}
	a.lock.Unlock()
	return bufs
}

// Undo allocation
func (a *Alloc) AbortNums(nums []uint64) {
	log.Printf("AbortBlks %v\n", nums)
	for _, n := range nums {
		a.Free(n)
	}
}
