package trans

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/txn"
)

type Trans struct {
	Fs     *fs.FsSuper
	txn    *txn.Txn
	balloc *Alloc
	ialloc *Alloc
	bufs   *buf.BufMap // map of bufs read/written by trans
	id     txn.TransId
}

func Begin(fs *fs.FsSuper, txn *txn.Txn, balloc *Alloc, ialloc *Alloc) *Trans {
	trans := &Trans{
		Fs:     fs,
		txn:    txn,
		balloc: balloc,
		ialloc: ialloc,
		bufs:   buf.MkBufMap(),
		id:     txn.GetTransId(),
	}
	return trans
}

func (trans *Trans) ReadBufLocked(addr buf.Addr) *buf.Buf {
	first := trans.txn.Lock(addr, trans.id)
	if first {
		buf := buf.MkBufData(addr)
		trans.txn.Load(buf)
		trans.bufs.Insert(buf)
	}
	b := trans.bufs.Lookup(addr)
	return b
}

func (trans *Trans) Release(addr buf.Addr) {
	trans.bufs.Del(addr)
	trans.txn.Release(addr, trans.id)
}

func (trans *Trans) NumberDirty() uint64 {
	return trans.bufs.Ndirty()
}

func (trans *Trans) LogSz() uint64 {
	return trans.txn.LogSz()
}

func (trans *Trans) LogSzBytes() uint64 {
	return trans.txn.LogSz() * disk.BlockSize
}

func (trans *Trans) AllocINum() fs.Inum {
	n := trans.ialloc.AllocNum(trans)
	return fs.Inum(n)
}

func (trans *Trans) FreeINum(inum fs.Inum) {
	trans.ialloc.FreeNum(trans, uint64(inum))
}

// Commit blocks of this transaction
func (trans *Trans) CommitWait(wait bool, abort bool) bool {
	return trans.txn.CommitWait(trans.bufs.Bufs(), wait, abort, trans.id)
}

func (trans *Trans) Flush() bool {
	return trans.txn.Flush(trans.id)
}
