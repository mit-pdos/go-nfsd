package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

// Returns a block
func loadBlock(slot *Cslot, a uint64) disk.Block {
	slot.lock()
	if slot.obj == nil {
		// blk hasn't been read yet from disk; read it and put
		// the buf with the read blk in the cache slot.
		blk := disk.Read(a)
		slot.obj = blk
	}
	blk := slot.obj.(disk.Block)
	slot.unlock()
	return blk
}

// If Read cannot find a cache slot, wait until installer unpins
// blocks from cache: flush memlog, which may contain unstable writes,
// and signal installer.
func (txn *Txn) readBlock(addr uint64) disk.Block {
	if addr >= txn.fs.Size {
		panic("Read")
	}
	var slot *Cslot
	slot = txn.bc.lookupSlot(addr)
	for slot == nil {
		log.Printf("ReadBlock: miss on %d WaitFlushMemLog and signal installer\n",
			addr)
		txn.log.WaitFlushMemLog()
		txn.log.SignalInstaller()
		if txn.amap.Len() >= txn.log.logSz {
			panic("readBlock")
		}
		// Try again; a slot should free up eventually.
		slot = txn.bc.lookupSlot(addr)
	}
	// load the block into the cache slot
	blk := loadBlock(slot, addr)
	return blk
}

func (txn *Txn) releaseBlock(blkno uint64) {
	txn.bc.freeSlot(blkno)
}

func (txn *Txn) AllocMyBlock(blkno uint64) uint64 {
	var n uint64 = 0
	bs := txn.amap.LookupBufs(blkno)
	for _, b := range bs {
		n = txn.balloc.Alloc1(b)
		if n != 0 {
			break
		}
	}
	return n
}

func (txn *Txn) AllocBlock() uint64 {
	var n uint64 = 0
	for i := txn.balloc.start; i < txn.balloc.start+txn.balloc.len; i++ {
		n = txn.AllocMyBlock(i)
		if n != 0 {
			break
		}

	}
	if n == 0 {
		b := txn.balloc.LockFreeRegion(txn)
		if b != nil {
			n = txn.balloc.Alloc1(b)
			if n == 0 {
				panic("AllocBlock")
			}
			b.Dirty()
		}
	}
	log.Printf("alloc block %v\n", n)
	return n
}

// XXX first check locally
func (txn *Txn) FreeBlock(blkno uint64) {
	log.Printf("free block %v\n", blkno)
	if blkno == 0 {
		panic("FreeBlock")
	}
	addr := txn.balloc.RegionAddr(blkno)
	var buf *Buf
	buf = txn.amap.Lookup(addr)
	if buf == nil {
		buf = txn.balloc.LockRegion(txn, blkno)
	}
	txn.balloc.Free1(buf, blkno)
	buf.Dirty()
}

func zeroBlock(txn *Txn, blkno uint64) {
	log.Printf("zero block %d\n", blkno)
	addr := txn.fs.Block2Addr(blkno)
	buf := txn.ReadBuf(addr)
	for i, _ := range buf.blk {
		buf.blk[i] = 0
	}
	buf.dirty = true
}
