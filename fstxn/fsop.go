package fstxn

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

type FsTxn struct {
	Fs     *fs.FsSuper
	buftxn *buftxn.BufTxn
	balloc *alloc.Alloc
	ialloc *alloc.Alloc
}

func Begin(fs *fs.FsSuper, txn *txn.Txn, balloc *alloc.Alloc, ialloc *alloc.Alloc) *FsTxn {
	op := &FsTxn{
		Fs:     fs,
		buftxn: buftxn.Begin(txn),
		balloc: balloc,
		ialloc: ialloc,
	}
	return op
}

func (op *FsTxn) NumberDirty() uint64 {
	return op.buftxn.NDirty()
}

func (op *FsTxn) LogSz() uint64 {
	return op.buftxn.LogSz()
}

func (op *FsTxn) LogSzBytes() uint64 {
	return op.buftxn.LogSz() * disk.BlockSize
}

// Commit bufs of this transaction
func (op *FsTxn) CommitWait(wait bool, abort bool) bool {
	return op.buftxn.CommitWait(wait, abort)
}

func (op *FsTxn) Flush() bool {
	return op.buftxn.Flush()
}

func (op *FsTxn) Release(addr buf.Addr) {
	op.buftxn.Release(addr)
}

func (op *FsTxn) ReadBufLocked(addr buf.Addr) *buf.Buf {
	return op.buftxn.ReadBufLocked(addr)
}

func (op *FsTxn) AllocINum() fs.Inum {
	n := op.ialloc.AllocNum(op.buftxn)
	return fs.Inum(n)
}

func (op *FsTxn) FreeINum(inum fs.Inum) {
	op.ialloc.FreeNum(op.buftxn, uint64(inum))
}

func (op *FsTxn) AllocBlock() uint64 {
	n := op.balloc.AllocNum(op.buftxn)
	util.DPrintf(5, "alloc block %v\n", n)
	return n
}

func (op *FsTxn) FreeBlock(blkno uint64) {
	util.DPrintf(5, "free block %v\n", blkno)
	op.balloc.FreeNum(op.buftxn, blkno)
}

func (op *FsTxn) ReadBlock(blkno uint64) *buf.Buf {
	util.DPrintf(10, "ReadBlock %d\n", blkno)
	addr := op.Fs.Block2addr(blkno)
	return op.buftxn.ReadBufLocked(addr)
}

func (op *FsTxn) ZeroBlock(blkno uint64) {
	util.DPrintf(5, "zero block %d\n", blkno)
	buf := op.ReadBlock(blkno)
	for i := range buf.Blk {
		buf.Blk[i] = 0
	}
	buf.SetDirty()
}
