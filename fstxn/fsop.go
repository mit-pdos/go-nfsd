package fstxn

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/addrlock"
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// fstxn implements transactions using buftxn.  It adds to buftxn
// support for (1) block and inode allocation and (2) an inode cache.
//

type FsState struct {
	Fs      *fs.FsSuper
	Txn     *txn.Txn
	Icache  *cache.Cache
	Balloc  *alloc.Alloc
	Ialloc  *alloc.Alloc
	BitLock *addrlock.LockMap
}

func MkFsState(fs *fs.FsSuper, txn *txn.Txn, icache *cache.Cache, balloc *alloc.Alloc, ialloc *alloc.Alloc, bitlock *addrlock.LockMap) *FsState {
	st := &FsState{
		Fs:      fs,
		Txn:     txn,
		Icache:  icache,
		Balloc:  balloc,
		Ialloc:  ialloc,
		BitLock: bitlock,
	}
	return st
}

type FsTxn struct {
	Fs     *fs.FsSuper
	buftxn *buftxn.BufTxn
	icache *cache.Cache
	balloc *alloc.Alloc
	ialloc *alloc.Alloc
	inums  []fs.Inum
}

func Begin(opstate *FsState) *FsTxn {
	op := &FsTxn{
		Fs:     opstate.Fs,
		buftxn: buftxn.Begin(opstate.Txn, opstate.BitLock),
		icache: opstate.Icache,
		balloc: opstate.Balloc,
		ialloc: opstate.Ialloc,
		inums:  make([]fs.Inum, 0),
	}
	return op
}

func (op *FsTxn) AddInum(inum fs.Inum) {
	op.inums = append(op.inums, inum)
}

func (op *FsTxn) OwnInum(inum fs.Inum) bool {
	var own = false
	for _, v := range op.inums {
		if v == inum {
			own = true
			break
		}
	}
	return own
}

func (op *FsTxn) DoneInum(inum fs.Inum) {
	for i, v := range op.inums {
		if v == inum {
			op.inums[i] = op.inums[len(op.inums)-1]
			op.inums = op.inums[:len(op.inums)-1]
		}
	}
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

func (op *FsTxn) Acquire(addr buf.Addr) {
	op.buftxn.Acquire(addr)
}

func (op *FsTxn) OwnLock(addr buf.Addr) bool {
	return op.buftxn.IsLocked(addr)
}

func (op *FsTxn) ReadBuf(addr buf.Addr) *buf.Buf {
	return op.buftxn.ReadBuf(addr)
}

func (op *FsTxn) OverWrite(addr buf.Addr, data []byte) {
	op.buftxn.OverWrite(addr, data)
}

func (op *FsTxn) LookupSlot(inum fs.Inum) *cache.Cslot {
	return op.icache.LookupSlot(uint64(inum))
}

func (op *FsTxn) FreeSlot(inum fs.Inum) {
	op.icache.FreeSlot(uint64(inum))
}

func (op *FsTxn) AllocINum() fs.Inum {
	n := op.ialloc.AllocNum(op.buftxn)
	return fs.Inum(n)
}

func (op *FsTxn) FreeINum(inum fs.Inum) {
	op.ialloc.FreeNum(op.buftxn, uint64(inum))
}

func (op *FsTxn) AssertValidBlock(blkno buf.Bnum) {
	if blkno > 0 && (blkno < op.Fs.DataStart() || blkno >= op.Fs.MaxBnum()) {
		panic("invalid blkno")
	}
}

func (op *FsTxn) AllocBlock() buf.Bnum {
	util.DPrintf(5, "alloc block\n")
	n := buf.Bnum(op.balloc.AllocNum(op.buftxn))
	op.AssertValidBlock(n)
	util.DPrintf(1, "alloc block -> %v\n", n)
	return n
}

func (op *FsTxn) FreeBlock(blkno buf.Bnum) {
	util.DPrintf(5, "free block %v\n", blkno)
	op.AssertValidBlock(blkno)
	if blkno == 0 {
		return
	}
	op.ZeroBlock(blkno)
	op.balloc.FreeNum(op.buftxn, uint64(blkno))
}

func (op *FsTxn) ReadBlock(blkno buf.Bnum) *buf.Buf {
	util.DPrintf(5, "ReadBlock %d\n", blkno)
	op.AssertValidBlock(blkno)
	addr := op.Fs.Block2addr(blkno)
	return op.buftxn.ReadBuf(addr)
}

func (op *FsTxn) ZeroBlock(blkno buf.Bnum) {
	util.DPrintf(5, "zero block %d\n", blkno)
	buf := op.ReadBlock(blkno)
	for i := range buf.Blk {
		buf.Blk[i] = 0
	}
	buf.SetDirty()
}
