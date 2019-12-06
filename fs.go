package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

type FsSuper struct {
	Size    uint64
	NLog    uint64 // including commit block
	NBitmap uint64
	NInode  uint64
	MaxAddr uint64
}

func mkFsSuper() *FsSuper {
	sz := uint64(10 * 10000)
	nbitmap := (sz / NBITS) + 1
	disk.Init(disk.NewMemDisk(sz))
	return &FsSuper{Size: sz,
		NLog:    LOGSIZE,
		NBitmap: nbitmap,
		NInode:  2000,
		MaxAddr: sz}
}

func (fs *FsSuper) bitmapStart() uint64 {
	return fs.NLog
}

func (fs *FsSuper) inodeStart() uint64 {
	return fs.NLog + fs.NBitmap
}

func (fs *FsSuper) dataStart() uint64 {
	return fs.NLog + fs.NBitmap + fs.NInode
}

//
// mkfs
//

func (fs *FsSuper) initFs() {
	nulli := mkNullInode() // inum = 0 is reserved
	nullblk := make(disk.Block, disk.BlockSize)
	nulli.encode(nullblk)
	fs.putBlkDirect(NULLINUM, nullblk)
	root := mkRootInode()
	rootblk := make(disk.Block, disk.BlockSize)
	root.encode(rootblk)
	fs.putBlkDirect(ROOTINUM, rootblk)
	fs.markAlloc(fs.inodeStart()+fs.NInode, fs.MaxAddr)
}

func (fs *FsSuper) markAlloc(n uint64, m uint64) {
	log.Printf("markAlloc: [0, %d) and [%d,%d)\n", n, m, fs.NBitmap*NBITS)
	if n >= NBITS || m >= NBITS*fs.NBitmap || m < NBITS {
		panic("markAlloc")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < n; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] |= 1 << bit
	}
	disk.Write(fs.bitmapStart(), blk)

	blk1 := make(disk.Block, disk.BlockSize)
	blkno := m/NBITS + fs.bitmapStart()
	for bn := m % disk.BlockSize; bn < disk.BlockSize; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] |= 1 << bit
	}
	disk.Write(blkno, blk1)
}

func (fs *FsSuper) putBlkDirect(inum uint64, blk disk.Block) bool {
	if inum >= fs.NInode {
		return false
	}
	log.Printf("write blk direct %d\n", fs.inodeStart()+inum)
	disk.Write(fs.inodeStart()+inum, blk)
	return true
}
