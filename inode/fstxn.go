package inode

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// fstxn implements transactions using buftxn.  It adds to buftxn
// support for (1) block and inode allocation and (2) an inode cache.
//

type FsTxn struct {
	Fs         *super.FsSuper
	buftxn     *buftxn.BufTxn
	icache     *cache.Cache
	balloc     *alloc.Alloc
	ialloc     *alloc.Alloc
	inodes     []*Inode
	allocInums []common.Inum
	freeInums  []common.Inum
	allocBnums []common.Bnum
	freeBnums  []common.Bnum
}

func Begin(opstate *FsState) *FsTxn {
	op := &FsTxn{
		Fs:         opstate.Fs,
		buftxn:     buftxn.Begin(opstate.Txn),
		icache:     opstate.Icache,
		balloc:     opstate.Balloc,
		ialloc:     opstate.Ialloc,
		inodes:     make([]*Inode, 0),
		allocInums: make([]common.Inum, 0),
		freeInums:  make([]common.Inum, 0),
		allocBnums: make([]common.Bnum, 0),
		freeBnums:  make([]common.Bnum, 0),
	}
	return op
}

func (op *FsTxn) addInode(ip *Inode) {
	op.inodes = append(op.inodes, ip)
}

func (op *FsTxn) OwnInum(inum common.Inum) bool {
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

func (op *FsTxn) AllocINum() common.Inum {
	n := common.Inum(op.ialloc.AllocNum())
	if n != common.NULLINUM {
		op.allocInums = append(op.allocInums, n)
	}
	util.DPrintf(1, "alloc inode -> # %v\n", n)
	return common.Inum(n)
}

func (op *FsTxn) FreeINum(inum common.Inum) {
	util.DPrintf(1, "free inode -> # %v\n", inum)
	op.freeInums = append(op.freeInums, inum)
}

// Write allocated bits to the on-disk bit maps
func (op *FsTxn) commitAlloc() {
	for _, inum := range op.allocInums {
		addr := addr.MkBitAddr(op.Fs.BitmapInodeStart(), uint64(inum))
		op.buftxn.OverWrite(addr, []byte{(1 << (inum % 8))})
	}
	for _, bn := range op.allocBnums {
		addr := addr.MkBitAddr(op.Fs.BitmapBlockStart(), uint64(bn))
		op.buftxn.OverWrite(addr, []byte{(1 << (bn % 8))})
	}
}

// On-disk bitmap has been updated; update in-memory state for free bits
func (op *FsTxn) commitFree() {
	for _, inum := range op.freeInums {
		op.ialloc.FreeNum(uint64(inum))
	}
	for _, bn := range op.freeBnums {
		op.balloc.FreeNum(bn)
	}
}

func (op *FsTxn) AssertValidBlock(blkno common.Bnum) {
	if blkno > 0 && (blkno < op.Fs.DataStart() || blkno >= op.Fs.MaxBnum()) {
		panic("invalid blkno")
	}
}

func (op *FsTxn) AllocBlock() common.Bnum {
	util.DPrintf(5, "alloc block\n")
	bn := common.Bnum(op.balloc.AllocNum())
	op.AssertValidBlock(bn)
	util.DPrintf(1, "alloc block -> %v\n", bn)
	if bn != common.NULLBNUM {
		op.allocBnums = append(op.allocBnums, bn)
	}
	return bn
}

func (op *FsTxn) FreeBlock(blkno common.Bnum) {
	util.DPrintf(1, "free block %v\n", blkno)
	op.AssertValidBlock(blkno)
	if blkno == 0 {
		return
	}
	op.ZeroBlock(blkno)
	op.freeBnums = append(op.freeBnums, blkno)
}

func (op *FsTxn) ReadBlock(blkno common.Bnum) *buf.Buf {
	util.DPrintf(5, "ReadBlock %d\n", blkno)
	op.AssertValidBlock(blkno)
	addr := op.Fs.Block2addr(blkno)
	return op.buftxn.ReadBuf(addr)
}

func (op *FsTxn) ZeroBlock(blkno common.Bnum) {
	util.DPrintf(5, "zero block %d\n", blkno)
	buf := op.ReadBlock(blkno)
	for i := range buf.Blk {
		buf.Blk[i] = 0
	}
	buf.SetDirty()
}
