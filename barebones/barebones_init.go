package barebones

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/util"
)

func markAlloc(super *super.FsSuper, n common.Bnum, m common.Bnum) {
	util.DPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m,
		super.NBlockBitmap*common.NBITBLOCK)
	if n >= common.Bnum(common.NBITBLOCK) ||
		m >= common.Bnum(common.NBITBLOCK*super.NBlockBitmap) ||
		m < n {
		panic("markAlloc: configuration makes no sense")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < uint64(n); bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] = blk[byte] | 1<<bit
	}
	super.Disk.Write(uint64(super.BitmapBlockStart()), blk)

	var blk1 = blk
	blkno := m/common.Bnum(common.NBITBLOCK) + super.BitmapBlockStart()
	if blkno > super.BitmapBlockStart() {
		blk1 = make(disk.Block, disk.BlockSize)
	}
	for bn := uint64(m) % common.NBITBLOCK; bn < common.NBITBLOCK; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk1[byte] = blk1[byte] | 1<<bit
	}
	super.Disk.Write(uint64(blkno), blk1)

	// mark inode 0 and 1 as allocated
	blk2 := make(disk.Block, disk.BlockSize)
	blk2[0] = blk2[0] | 1<<0
	blk2[0] = blk2[0] | 1<<1
	super.Disk.Write(uint64(super.BitmapInodeStart()), blk2)
}

func makeFs(super *super.FsSuper) {
	util.DPrintf(1, "mkfs")

	root := inode.MkRootInode()
	util.DPrintf(1, "root %v\n", root)
	raddr := super.Inum2Addr(common.ROOTINUM)
	rootblk := root.Encode()
	rootbuf := buf.MkBuf(raddr, common.INODESZ*8, rootblk)
	rootbuf.WriteDirect(super.Disk)

	markAlloc(super, super.DataStart(), super.MaxBnum())
}

func (nfs *BarebonesNfs) readBitmap() {
	var bitmap []byte
	start := nfs.fs.BitmapInodeStart()
	for i := uint64(0); i < nfs.fs.NInodeBitmap; i++ {
		blk := nfs.fs.Disk.Read(uint64(start) + i)
		bitmap = append(bitmap, blk...)
	}
	nfs.bitmap = bitmap
}

func MkNfs(d disk.Disk) *BarebonesNfs {
	return &BarebonesNfs{
		glocks: nil,
		fs: super.MkFsSuper(d),
		txn: nil,
		bitmap: nil,
	}
}

func (nfs *BarebonesNfs) InitLocks() {
	nfs.glocks = lockmap.MkLockMap()
}

func (nfs *BarebonesNfs) InitTxn() {
	nfs.txn = txn.MkTxn(nfs.fs) // runs recovery
}

func (nfs *BarebonesNfs) InitFs() {
	makeFs(nfs.fs)
	nfs.readBitmap()
}

func (nfs *BarebonesNfs) Init() {
	nfs.InitLocks()
	nfs.InitTxn()
	nfs.InitFs()
}

