package inode

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// fstxn implements transactions using buftxn.  It adds to buftxn
// support for (1) block and inode allocation and (2) an inode cache.
//

type FsTxn struct {
	Fs     *fs.FsSuper
	buftxn *buftxn.BufTxn
	icache *cache.Cache
	balloc *alloc.Alloc
	ialloc *alloc.Alloc
	inodes []*Inode
}

func Begin(opstate *FsState) *FsTxn {
	op := &FsTxn{
		Fs:     opstate.Fs,
		buftxn: buftxn.Begin(opstate.Txn, opstate.BitLock),
		icache: opstate.Icache,
		balloc: opstate.Balloc,
		ialloc: opstate.Ialloc,
		inodes: make([]*Inode, 0),
	}
	return op
}

func (op *FsTxn) addInode(ip *Inode) {
	op.inodes = append(op.inodes, ip)
}

func (op *FsTxn) OwnInum(inum fs.Inum) bool {
	var own = false
	for _, ip := range op.inodes {
		if ip.Inum == inum {
			own = true
			break
		}
	}
	return own
}

func (op *FsTxn) doneInode(ip *Inode) {
	for i, v := range op.inodes {
		if v == ip {
			op.inodes[i] = op.inodes[len(op.inodes)-1]
			op.inodes = op.inodes[:len(op.inodes)-1]
		}
	}
}

func putInodes(op *FsTxn) {
	for _, ip := range op.inodes {
		ip.Put(op)
	}
}

func releaseInodes(op *FsTxn) {
	for _, ip := range op.inodes {
		ip.ReleaseInode(op)
	}
}

func (op *FsTxn) LogSzBytes() uint64 {
	return op.buftxn.LogSz() * disk.BlockSize
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
