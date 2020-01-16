package wal

import (
	"github.com/tchajed/goose/machine/disk"
)

const (
	NBITBLOCK    uint64 = disk.BlockSize * 8
	INODEBLK     uint64 = disk.BlockSize / INODESZ
	NINODEBITMAP uint64 = 1

	INODESZ uint64 = 64 // on-disk size

	HDRMETA  = uint64(8) // space for the end position
	HDRADDRS = (disk.BlockSize - HDRMETA) / 8
	LOGSIZE  = HDRADDRS + 2 // 2 for log header
)

type Inum uint64

const NULLINUM Inum = 0
const ROOTINUM Inum = 1

type FsSuper struct {
	Size         uint64
	nLog         uint64 // including commit block
	NBlockBitmap uint64
	NInodeBitmap uint64
	nInodeBlk    uint64
	Maxaddr      uint64
}

func (fs *FsSuper) BitmapBlockStart() uint64 {
	return fs.nLog
}

func (fs *FsSuper) BitmapInodeStart() uint64 {
	return fs.BitmapBlockStart() + fs.NBlockBitmap
}

func (fs *FsSuper) InodeStart() uint64 {
	return fs.BitmapInodeStart() + fs.NInodeBitmap
}

func (fs *FsSuper) DataStart() uint64 {
	return fs.InodeStart() + fs.nInodeBlk
}

// XXX goose emits something nonsensical
//func (fs *FsSuper) NInode() Inum {
//	return Inum(fs.nInodeBlk * INODEBLK)
//}
