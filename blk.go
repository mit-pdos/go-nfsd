package goose_nfs

func (txn *Txn) AllocBlock() uint64 {
	n := txn.balloc.AllocNum(txn)
	DPrintf("alloc block %v\n", n)
	return n
}

func (txn *Txn) FreeBlock(blkno uint64) {
	DPrintf("free block %v\n", blkno)
	txn.balloc.FreeNum(txn, blkno)
}

func ReadBlock(txn *Txn, blkno uint64) *Buf {
	// DPrintf("ReadBlock %d\n", blkno)
	addr := txn.fs.Block2Addr(blkno)
	return txn.ReadBufLocked(addr, BLOCK)
}

func ZeroBlock(txn *Txn, blkno uint64) {
	DPrintf("zero block %d\n", blkno)
	buf := ReadBlock(txn, blkno)
	for i, _ := range buf.blk {
		buf.blk[i] = 0
	}
	buf.dirty = true
}
