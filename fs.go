package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"
)

const (
	NBITBLOCK    uint64 = disk.BlockSize * 8
	INODEBLK     uint64 = disk.BlockSize / INODESZ
	NINODEBITMAP uint64 = 1
)

type FsSuper struct {
	Size         uint64
	NLog         uint64 // including commit block
	NBlockBitmap uint64
	NInodeBitmap uint64
	NInodeBlk    uint64
	MaxAddr      uint64
}

func mkFsSuper() *FsSuper {
	sz := uint64(10 * 10000)
	nblockbitmap := (sz / NBITBLOCK) + 1
	disk.Init(disk.NewMemDisk(sz))
	return &FsSuper{
		Size:         sz,
		NLog:         LOGSIZE,
		NBlockBitmap: nblockbitmap,
		NInodeBitmap: NINODEBITMAP,
		NInodeBlk:    (NINODEBITMAP * NBITBLOCK * INODESZ) / disk.BlockSize,
		MaxAddr:      sz}
}

func (fs *FsSuper) bitmapBlockStart() uint64 {
	return fs.NLog
}

func (fs *FsSuper) bitmapInodeStart() uint64 {
	return fs.bitmapBlockStart() + fs.NBlockBitmap
}

func (fs *FsSuper) inodeStart() uint64 {
	return fs.bitmapInodeStart() + fs.NInodeBitmap
}

func (fs *FsSuper) dataStart() uint64 {
	return fs.inodeStart() + fs.NInodeBlk
}

func (fs *FsSuper) Block2Addr(blkno uint64) Addr {
	return mkAddr(blkno, 0, disk.BlockSize)
}

func (fs *FsSuper) NInode() uint64 {
	return fs.NInodeBlk * INODEBLK
}

func (fs *FsSuper) Inum2Addr(inum Inum) Addr {
	return mkAddr(fs.inodeStart()+inum/INODEBLK, (inum%INODEBLK)*INODESZ, INODESZ)
}

//
// mkfs
//

func (fs *FsSuper) initFs() {
	// inum = 0 is reserved
	nulli := mkNullInode()
	naddr := fs.Inum2Addr(NULLINUM)
	nullblk := make(disk.Block, INODESZ)
	buf := mkBuf(naddr, 0, nullblk, nil)
	nulli.encode(buf)
	buf.WriteDirect()

	root := mkRootInode()
	DPrintf(5, "root %v\n", root)
	raddr := fs.Inum2Addr(ROOTINUM)
	rootblk := make(disk.Block, INODESZ)
	rootbuf := mkBuf(raddr, 0, rootblk, nil)
	root.encode(rootbuf)
	rootbuf.WriteDirect()

	fs.markAlloc(fs.dataStart(), fs.MaxAddr)
}

func (fs *FsSuper) markAlloc(n uint64, m uint64) {
	DPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m, fs.NBlockBitmap*NBITBLOCK)
	if n >= NBITBLOCK || m >= NBITBLOCK*fs.NBlockBitmap || m < NBITBLOCK {
		panic("markAlloc")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < n; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] |= 1 << bit
	}
	disk.Write(fs.bitmapBlockStart(), blk)

	blk1 := make(disk.Block, disk.BlockSize)
	blkno := m/NBITBLOCK + fs.bitmapBlockStart()
	for bn := m % disk.BlockSize; bn < NBITBLOCK; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk1[byte] |= 1 << bit
	}
	disk.Write(blkno, blk1)

	// mark inode 0 and 1 as allocated
	blk2 := make(disk.Block, disk.BlockSize)
	blk2[0] |= (1 << 0)
	blk2[0] |= (1 << 1)
	disk.Write(fs.bitmapInodeStart(), blk2)
}

func (fs *FsSuper) putBlkDirect(inum uint64, blk disk.Block) bool {
	if inum >= fs.NInode() {
		return false
	}
	DPrintf(10, "write blk direct %d\n", fs.inodeStart()+inum)
	disk.Write(fs.inodeStart()+inum, blk)
	return true
}
