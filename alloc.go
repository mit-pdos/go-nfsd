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
}

func mkAlloc(start uint64, len uint64) *Alloc {
	bmap := make([]disk.Block, len)
	bmapCommit := make([]disk.Block, len)
	for i := uint64(0); i < len; i++ {
		blkno := start + i
		bmap[i] = disk.Read(blkno)
		bmapCommit[i] = disk.Read(blkno)
	}
	a := &Alloc{
		lock:       new(sync.RWMutex),
		bmap:       bmap,
		bmapCommit: bmapCommit,
		start:      start,
		len:        len,
	}
	return a
}

// Find a free bit in blk and toggle it
func findAndMark(blk disk.Block) (uint64, bool) {
	for byte := uint64(0); byte < disk.BlockSize; byte++ {
		byteVal := blk[byte]
		if byteVal == 0xff {
			continue
		}
		for bit := uint64(0); bit < 8; bit++ {
			if byteVal&(1<<bit) == 0 {
				off := 8*byte + bit
				markBit(blk, off)
				return off, true
			}
		}
	}
	return 0, false
}

// Free bit bn in blk
func freeBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] = blk[byte] & ^(1 << bit)
}

// Alloc bit bn in blk
func markBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] |= (1 << bit)
}

func (a *Alloc) markBlock(bn uint64) {
	i := bn / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}

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

func (a *Alloc) Free(n uint64) {
	a.lock.Lock()
	i := n / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}
	freeBit(a.bmap[i], n%NBITBLOCK)
	a.lock.Unlock()
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
