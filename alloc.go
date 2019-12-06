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
	fs         *FsSuper
	bmapCommit []disk.Block
}

func mkAlloc(fs *FsSuper) *Alloc {
	bmap := make([]disk.Block, fs.NBitmap)
	bmapCommit := make([]disk.Block, fs.NBitmap)
	for i := uint64(0); i < fs.NBitmap; i++ {
		blkno := fs.bitmapStart() + i
		bmap[i] = disk.Read(blkno)
		bmapCommit[i] = disk.Read(blkno)
	}
	a := &Alloc{
		lock:       new(sync.RWMutex),
		bmap:       bmap,
		bmapCommit: bmapCommit,
		fs:         fs,
	}
	return a
}

const NBITS uint64 = disk.BlockSize * 8

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
	i := bn / NBITS
	if i >= a.fs.NBitmap {
		panic("freeBlock")
	}

}

// Zero indicates failure
func (a *Alloc) AllocBlock() uint64 {
	var bit uint64 = 0

	a.lock.Lock()
	for i := uint64(0); i < a.fs.NBitmap; i++ {
		b, found := findAndMark(a.bmap[i])
		if !found {
			continue
		}
		bit = i*NBITS + b
		break
	}
	a.lock.Unlock()
	return bit
}

func (a *Alloc) FreeBlock(bn uint64) {
	a.lock.Lock()
	i := bn / NBITS
	if i >= a.fs.NBitmap {
		panic("freeBlock")
	}
	freeBit(a.bmap[i], bn%NBITS)
	a.lock.Unlock()
}

func (a *Alloc) CommitBmap(alloc []uint64, free []uint64) []*Buf {
	bufs := make([]*Buf, 0)
	dirty := make([]bool, a.fs.NBitmap)
	a.lock.Lock()
	for _, bn := range alloc {
		i := bn / NBITS
		dirty[i] = true
		markBit(a.bmapCommit[i], bn%NBITS)
	}
	for _, bn := range free {
		i := bn / NBITS
		dirty[i] = true
		freeBit(a.bmap[i], bn%NBITS)
		freeBit(a.bmapCommit[i], bn%NBITS)
	}
	for i, v := range dirty {
		if v {
			buf := mkBuf(a.fs.bitmapStart()+uint64(i), a.bmapCommit[i])
			bufs = append(bufs, buf)
		}
	}
	a.lock.Unlock()
	return bufs
}

// Undo allocation
func (a *Alloc) AbortBlks(blknos []uint64) {
	log.Printf("AbortBlks %v\n", blknos)
	for _, bn := range blknos {
		a.FreeBlock(bn)
	}
}
