package fs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/util"
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
	Disk         disk.Disk
	Size         uint64
	nLog         uint64 // including commit block
	NBlockBitmap uint64
	NInodeBitmap uint64
	nInodeBlk    uint64
	Maxaddr      uint64
}

func MkFsSuper(sz uint64, name *string) *FsSuper {
	nblockbitmap := (sz / NBITBLOCK) + 1
	var d disk.Disk
	if name != nil {
		util.DPrintf(0, "MkFsSuper: create file disk %s\n", *name)
		file, err := disk.NewFileDisk(*name, sz)
		if err != nil {
			panic("MkFsSuper: couldn't create disk image")
		}
		d = file
	} else {
		util.DPrintf(0, "MkFsSuper: create mem disk\n")
		d = disk.NewMemDisk(sz)
	}

	return &FsSuper{
		Disk:         d,
		Size:         sz,
		nLog:         LOGSIZE,
		NBlockBitmap: nblockbitmap,
		NInodeBitmap: NINODEBITMAP,
		nInodeBlk:    (NINODEBITMAP * NBITBLOCK * INODESZ) / disk.BlockSize,
		Maxaddr:      sz}
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

func (fs *FsSuper) Block2addr(blkno uint64) buf.Addr {
	return buf.MkAddr(blkno, 0, NBITBLOCK)
}

func (fs *FsSuper) NInode() Inum {
	return Inum(fs.nInodeBlk * INODEBLK)
}

func (fs *FsSuper) Inum2Addr(inum Inum) buf.Addr {
	return buf.MkAddr(fs.InodeStart()+uint64(inum)/INODEBLK, (uint64(inum)%INODEBLK)*INODESZ*8, INODESZ*8)
}

func (fs *FsSuper) DiskBlockSize(addr buf.Addr) bool {
	return addr.Sz == NBITBLOCK
}
