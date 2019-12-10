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

func (txn *Txn) AllocBlock() uint64 {
	n := txn.balloc.AllocNum(txn)
	log.Printf("alloc block %v\n", n)
	return n
}

func (txn *Txn) FreeBlock(blkno uint64) {
	log.Printf("free block %v\n", blkno)
	txn.balloc.FreeNum(txn, blkno)
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
