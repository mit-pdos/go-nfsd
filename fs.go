package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"
)

const (
	NBITBLOCK    uint64 = disk.BlockSize * 8
	INODEBLK     uint64 = disk.BlockSize / INODESZ
	NINODEBITMAP uint64 = 1
)

type fsSuper struct {
	size         uint64
	nLog         uint64 // including commit block
	nBlockBitmap uint64
	nInodeBitmap uint64
	nInodeBlk    uint64
	maxaddr      uint64
}

func mkFsSuper() *fsSuper {
	sz := uint64(10 * 10000)
	nblockbitmap := (sz / NBITBLOCK) + 1
	disk.Init(disk.NewMemDisk(sz))
	return &fsSuper{
		size:         sz,
		nLog:         LOGSIZE,
		nBlockBitmap: nblockbitmap,
		nInodeBitmap: NINODEBITMAP,
		nInodeBlk:    (NINODEBITMAP * NBITBLOCK * INODESZ) / disk.BlockSize,
		maxaddr:      sz}
}

func (fs *fsSuper) bitmapBlockStart() uint64 {
	return fs.nLog
}

func (fs *fsSuper) bitmapInodeStart() uint64 {
	return fs.bitmapBlockStart() + fs.nBlockBitmap
}

func (fs *fsSuper) inodeStart() uint64 {
	return fs.bitmapInodeStart() + fs.nInodeBitmap
}

func (fs *fsSuper) dataStart() uint64 {
	return fs.inodeStart() + fs.nInodeBlk
}

func (fs *fsSuper) block2addr(blkno uint64) addr {
	return mkaddr(blkno, 0, NBITBLOCK)
}

func (fs *fsSuper) nInode() inum {
	return inum(fs.nInodeBlk * INODEBLK)
}

func (fs *fsSuper) inum2addr(inum inum) addr {
	return mkaddr(fs.inodeStart()+uint64(inum)/INODEBLK, (uint64(inum)%INODEBLK)*INODESZ*8, INODESZ*8)
}

//
// mkfs
//

func (fs *fsSuper) initFs() {
	// inum = 0 is reserved
	nulli := mkNullInode()
	naddr := fs.inum2addr(NULLINUM)
	nullblk := make(disk.Block, INODESZ)
	buf := mkBuf(naddr, 0, nullblk, nil)
	nulli.encode(buf)
	buf.writeDirect()

	root := mkRootInode()
	dPrintf(5, "root %v\n", root)
	raddr := fs.inum2addr(ROOTINUM)
	rootblk := make(disk.Block, INODESZ)
	rootbuf := mkBuf(raddr, 0, rootblk, nil)
	root.encode(rootbuf)
	rootbuf.writeDirect()

	fs.markAlloc(fs.dataStart(), fs.maxaddr)
}

func (fs *fsSuper) markAlloc(n uint64, m uint64) {
	dPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m, fs.nBlockBitmap*NBITBLOCK)
	if n >= NBITBLOCK || m >= NBITBLOCK*fs.nBlockBitmap || m < NBITBLOCK {
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
