package goose_nfs

func (txn *txn) allocBlock() uint64 {
	n := txn.balloc.allocNum(txn)
	dPrintf(5, "alloc block %v\n", n)
	return n
}

func (txn *txn) freeBlock(blkno uint64) {
	dPrintf(5, "free block %v\n", blkno)
	txn.balloc.freeNum(txn, blkno)
}

func (txn *txn) readBlock(blkno uint64) *buf {
	dPrintf(10, "ReadBlock %d\n", blkno)
	addr := txn.fs.block2addr(blkno)
	return txn.readBufLocked(addr, BLOCK)
}

func (txn *txn) zeroBlock(blkno uint64) {
	dPrintf(5, "zero block %d\n", blkno)
	buf := txn.readBlock(blkno)
	for i, _ := range buf.blk {
		buf.blk[i] = 0
	}
	buf.dirty = true
}
