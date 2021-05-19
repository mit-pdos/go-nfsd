package inode

import (
	"github.com/mit-pdos/go-journal/jrnl"
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/goose-nfsd/alloctxn"
)

//
// Freeing of a file. Freeing a large file may require multiple
// transactions to ensure that the indirect blocks modified due to a
// free fit in the write-ahead log.  In this case the caller of
// Shrink() is responsible for starting another shrink transaction.
//

func (ip *Inode) shrinkFits(op *alloctxn.AllocTxn, nblk uint64) bool {
	return op.Op.NDirty()+nblk < jrnl.LogBlocks
}

func (ip *Inode) IsShrinking() bool {
	cursz := util.RoundUp(ip.Size, disk.BlockSize)
	s := ip.ShrinkSize > cursz
	return s
}

func (ip *Inode) freeIndex(op *alloctxn.AllocTxn, index uint64) {
	op.FreeBlock(ip.blks[index])
	ip.blks[index] = 0
}

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *Inode) indshrink(op *alloctxn.AllocTxn, root common.Bnum, level uint64, bn uint64) common.Bnum {
	if root == common.NULLBNUM {
		return 0
	}
	if level == 0 {
		return root
	}
	divisor := pow(level - 1)
	off := (bn / divisor)
	ind := bn % divisor
	boff := off * 8
	b := op.ReadBlock(root)
	nxtroot := b.BnumGet(boff)
	op.AssertValidBlock(nxtroot)
	if nxtroot != 0 {
		freeroot := ip.indshrink(op, nxtroot, level-1, ind)
		if freeroot != 0 {
			b.BnumPut(boff, 0)
			op.FreeBlock(freeroot)
		}
	}
	if off == 0 && ind == 0 {
		return root
	} else {
		return common.NULLBNUM
	}
}

// Frees as many blocks as possible, and returns if more shrinking is necessary.
// 5: inode block, 2xbitmap block, indirect block, double indirect
func (ip *Inode) Shrink(op *alloctxn.AllocTxn) bool {
	util.DPrintf(1, "Shrink: from %d to %d\n", ip.ShrinkSize,
		util.RoundUp(ip.Size, disk.BlockSize))
	for ip.IsShrinking() && ip.shrinkFits(op, 5) {
		ip.ShrinkSize -= 1
		if ip.ShrinkSize < NDIRECT {
			ip.freeIndex(op, ip.ShrinkSize)
		} else {
			var off = ip.ShrinkSize - NDIRECT
			if off < NBLKBLK {
				freeroot := ip.indshrink(op, ip.blks[INDIRECT], 1, off)
				if freeroot != 0 {
					ip.freeIndex(op, INDIRECT)
				}
			} else {
				off = off - NBLKBLK
				freeroot := ip.indshrink(op, ip.blks[DINDIRECT], 2, off)
				if freeroot != 0 {
					ip.freeIndex(op, DINDIRECT)
				}
			}
		}
	}
	ip.WriteInode(op)
	return ip.IsShrinking()
}
