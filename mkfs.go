package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// mkfs
//

func initFs(super *fs.FsSuper) {
	// inum = 0 is reserved
	nulli := inode.MkNullInode()
	naddr := super.Inum2Addr(fs.NULLINUM)
	nullblk := make(disk.Block, fs.INODESZ)
	b := buf.MkBuf(naddr, nullblk)
	nulli.Encode(b)
	b.WriteDirect()

	root := inode.MkRootInode()
	util.DPrintf(5, "root %v\n", root)
	raddr := super.Inum2Addr(fs.ROOTINUM)
	rootblk := make(disk.Block, fs.INODESZ)
	rootbuf := buf.MkBuf(raddr, rootblk)
	root.Encode(rootbuf)
	rootbuf.WriteDirect()

	markAlloc(super, super.DataStart(), super.Maxaddr)
}

func markAlloc(super *fs.FsSuper, n uint64, m uint64) {
	util.DPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m,
		super.NBlockBitmap*alloc.NBITBLOCK)
	if n >= alloc.NBITBLOCK || m >= alloc.NBITBLOCK*super.NBlockBitmap || m < alloc.NBITBLOCK {
		panic("markAlloc")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < n; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] = blk[byte] | 1<<bit
	}
	disk.Write(super.BitmapBlockStart(), blk)

	blk1 := make(disk.Block, disk.BlockSize)
	blkno := m/alloc.NBITBLOCK + super.BitmapBlockStart()
	for bn := m % disk.BlockSize; bn < alloc.NBITBLOCK; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk1[byte] = blk1[byte] | 1<<bit
	}
	disk.Write(blkno, blk1)

	// mark inode 0 and 1 as allocated
	blk2 := make(disk.Block, disk.BlockSize)
	blk2[0] = blk2[0] | 1<<0
	blk2[0] = blk2[0] | 1<<1
	disk.Write(super.BitmapInodeStart(), blk2)
}
