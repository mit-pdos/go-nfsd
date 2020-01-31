package bcache

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/cache"
)

//
// Write-through block cache
//

const BCACHESZ uint64 = 512

type Bcache struct {
	// d      disk.Disk
	bcache *cache.Cache
}

func MkBcache() *Bcache {
	return &Bcache{
		bcache: cache.MkCache(BCACHESZ),
	}
}

func (bc *Bcache) Read(bn uint64) disk.Block {
	cslot := bc.bcache.LookupSlot(bn)
	if cslot == nil {
		panic("readBlock")
	}
	cslot.Lock()
	if cslot.Obj == nil {
		cslot.Obj = disk.Read(bn)
	}
	b := cslot.Obj.(disk.Block)
	blk := make([]byte, disk.BlockSize)
	copy(blk, b)
	cslot.Unlock()
	bc.bcache.FreeSlot(bn)
	return blk
}

func (bc *Bcache) Write(bn uint64, b disk.Block) {
	if b == nil {
		panic("Write")
	}
	cslot := bc.bcache.LookupSlot(bn)
	if cslot != nil {
		cslot.Lock()
		cslot.Obj = b
		cslot.Unlock()
		bc.bcache.FreeSlot(bn)
	}
	disk.Write(bn, b)
}

// func (bc *Bcache) Close() {
// 	bc.d.Close()
// }

func (bc *Bcache) Barrier() {
	disk.Barrier()
}

func (bc *Bcache) Size() uint64 {
	return disk.Size()
}
