package goose_nfs

import (
	"log"
)

func (txn *Txn) AllocBlock() uint64 {
	n := txn.balloc.AllocNum(txn)
	log.Printf("alloc block %v\n", n)
	return n
}

func (txn *Txn) FreeBlock(blkno uint64) {
	log.Printf("free block %v\n", blkno)
	txn.balloc.FreeNum(txn, blkno)
}

func ReadBlock(txn *Txn, blkno uint64) *Buf {
	// log.Printf("ReadBlock %d\n", blkno)
	addr := txn.fs.Block2Addr(blkno)
	return txn.ReadBufLocked(addr)
}

func ZeroBlock(txn *Txn, blkno uint64) {
	log.Printf("zero block %d\n", blkno)
	buf := ReadBlock(txn, blkno)
	for i, _ := range buf.blk {
		buf.blk[i] = 0
	}
	buf.dirty = true
}
