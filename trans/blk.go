package trans

import (
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/util"
)

func (trans *Trans) AllocBlock() uint64 {
	n := trans.balloc.AllocNum(trans)
	util.DPrintf(5, "alloc block %v\n", n)
	return n
}

func (trans *Trans) FreeBlock(blkno uint64) {
	util.DPrintf(5, "free block %v\n", blkno)
	trans.balloc.FreeNum(trans, blkno)
}

func (trans *Trans) ReadBlock(blkno uint64) *buf.Buf {
	util.DPrintf(10, "ReadBlock %d\n", blkno)
	addr := trans.Fs.Block2addr(blkno)
	return trans.ReadBufLocked(addr)
}

func (trans *Trans) ZeroBlock(blkno uint64) {
	util.DPrintf(5, "zero block %d\n", blkno)
	buf := trans.ReadBlock(blkno)
	for i := range buf.Blk {
		buf.Blk[i] = 0
	}
	buf.SetDirty()
}
