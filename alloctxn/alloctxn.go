package alloctxn

import (
	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// alloctxn implements transactions using buftxn.  It adds to buftxn
// support for (1) block and inode allocation.
//

type AllocTxn struct {
	Super      *super.FsSuper
	Buftxn     *buftxn.BufTxn
	Balloc     *alloc.Alloc
	Ialloc     *alloc.Alloc
	allocInums []common.Inum
	freeInums  []common.Inum
	allocBnums []common.Bnum
	freeBnums  []common.Bnum
}

func Begin(super *super.FsSuper, txn *txn.Txn, balloc *alloc.Alloc, ialloc *alloc.Alloc) *AllocTxn {
	atxn := &AllocTxn{
		Super:      super,
		Buftxn:     buftxn.Begin(txn),
		Ialloc:     ialloc,
		Balloc:     balloc,
		allocInums: make([]common.Inum, 0),
		freeInums:  make([]common.Inum, 0),
		allocBnums: make([]common.Bnum, 0),
		freeBnums:  make([]common.Bnum, 0),
	}
	return atxn
}

func (atxn *AllocTxn) Id() txn.TransId {
	return atxn.Buftxn.Id
}

func (atxn *AllocTxn) AllocINum() common.Inum {
	inum := common.Inum(atxn.Ialloc.AllocNum())
	util.DPrintf(1, "AllocINum -> # %v\n", inum)
	if inum != common.NULLINUM {
		atxn.allocInums = append(atxn.allocInums, inum)
	}
	return inum
}

func (atxn *AllocTxn) FreeINum(inum common.Inum) {
	util.DPrintf(1, "FreeINum -> # %v\n", inum)
	atxn.freeInums = append(atxn.freeInums, inum)
}

func (atxn *AllocTxn) WriteBits(nums []uint64, blk uint64, alloc bool) {
	for _, n := range nums {
		a := addr.MkBitAddr(blk, n)
		var b = byte(1 << (n % 8))
		if !alloc {
			b = ^b
		}
		atxn.Buftxn.OverWrite(a, 1, []byte{b})
	}
}

// Write allocated/free bits to the on-disk bit maps
func (atxn *AllocTxn) PreCommit() {
	util.DPrintf(1, "commitBitmaps: alloc inums %v blks %v\n", atxn.allocInums,
		atxn.allocBnums)

	atxn.WriteBits(atxn.allocInums, atxn.Super.BitmapInodeStart(), true)
	atxn.WriteBits(atxn.allocBnums, atxn.Super.BitmapBlockStart(), true)

	util.DPrintf(1, "commitBitmaps: free inums %v blks %v\n", atxn.freeInums,
		atxn.freeBnums)

	atxn.WriteBits(atxn.freeInums, atxn.Super.BitmapInodeStart(), false)
	atxn.WriteBits(atxn.freeBnums, atxn.Super.BitmapBlockStart(), false)
}

// On-disk bitmap has been updated; update in-memory state for free bits
func (atxn *AllocTxn) PostCommit() {
	util.DPrintf(1, "updateFree: inums %v blks %v\n", atxn.freeInums, atxn.freeBnums)
	for _, inum := range atxn.freeInums {
		atxn.Ialloc.FreeNum(uint64(inum))
	}
	for _, bn := range atxn.freeBnums {
		atxn.Balloc.FreeNum(bn)
	}
}

// Abort: free allocated inums and bnums. Nothing to do for freed
// ones, because in-memory state hasn't been updated by freeINum()/freeBlock().
func (atxn *AllocTxn) PostAbort() {
	util.DPrintf(1, "Abort: inums %v blks %v\n", atxn.allocInums, atxn.allocBnums)
	for _, inum := range atxn.allocInums {
		atxn.Ialloc.FreeNum(uint64(inum))
	}
	for _, bn := range atxn.allocBnums {
		atxn.Balloc.FreeNum(bn)
	}
}

func (atxn *AllocTxn) AssertValidBlock(blkno common.Bnum) {
	if blkno > 0 && (blkno < atxn.Super.DataStart() ||
		blkno >= atxn.Super.MaxBnum()) {
		panic("invalid blkno")
	}
}

func (atxn *AllocTxn) AllocBlock() common.Bnum {
	util.DPrintf(5, "alloc block\n")
	bn := common.Bnum(atxn.Balloc.AllocNum())
	atxn.AssertValidBlock(bn)
	util.DPrintf(1, "alloc block -> %v\n", bn)
	if bn != common.NULLBNUM {
		atxn.allocBnums = append(atxn.allocBnums, bn)
	}
	return bn
}

func (atxn *AllocTxn) FreeBlock(blkno common.Bnum) {
	util.DPrintf(1, "free block %v\n", blkno)
	atxn.AssertValidBlock(blkno)
	if blkno == 0 {
		return
	}
	atxn.ZeroBlock(blkno)
	atxn.freeBnums = append(atxn.freeBnums, blkno)
}

func (atxn *AllocTxn) ReadBlock(blkno common.Bnum) *buf.Buf {
	util.DPrintf(5, "ReadBlock %d\n", blkno)
	atxn.AssertValidBlock(blkno)
	addr := atxn.Super.Block2addr(blkno)
	return atxn.Buftxn.ReadBuf(addr, common.NBITBLOCK)
}

func (atxn *AllocTxn) ZeroBlock(blkno common.Bnum) {
	util.DPrintf(5, "zero block %d\n", blkno)
	buf := atxn.ReadBlock(blkno)
	for i := range buf.Data {
		buf.Data[i] = 0
	}
	buf.SetDirty()
}
