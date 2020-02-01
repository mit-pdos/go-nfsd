package fs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/fake-bcache/bcache"
	"github.com/mit-pdos/goose-nfsd/util"
	"github.com/mit-pdos/goose-nfsd/wal"
)

const (
	NBITBLOCK    uint64 = disk.BlockSize * 8
	INODEBLK     uint64 = disk.BlockSize / INODESZ
	NINODEBITMAP uint64 = 1

	INODESZ uint64 = 128 // on-disk size
)

type Inum uint64

const NULLINUM Inum = 0
const ROOTINUM Inum = 1

type FsSuper struct {
	Disk         *bcache.Bcache
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
		util.DPrintf(1, "MkFsSuper: open file disk %s\n", *name)
		file, err := disk.NewFileDisk(*name, sz)
		if err != nil {
			panic("MkFsSuper: couldn't create disk image")
		}
		d = file
	} else {
		util.DPrintf(1, "MkFsSuper: create mem disk\n")
		d = disk.NewMemDisk(sz)
	}

	// use the disk with a buffer cache
	disk.Init(d)
	bc := bcache.MkBcache()

	return &FsSuper{
		Disk:         bc,
		Size:         sz,
		nLog:         wal.LOGDISKBLOCKS,
		NBlockBitmap: nblockbitmap,
		NInodeBitmap: NINODEBITMAP,
		nInodeBlk:    (NINODEBITMAP * NBITBLOCK * INODESZ) / disk.BlockSize,
		Maxaddr:      sz}
}

func (fs *FsSuper) MaxBnum() buf.Bnum {
	return buf.Bnum(fs.Maxaddr)
}

func (fs *FsSuper) BitmapBlockStart() buf.Bnum {
	return buf.Bnum(fs.nLog)
}

func (fs *FsSuper) BitmapInodeStart() buf.Bnum {
	return fs.BitmapBlockStart() + buf.Bnum(fs.NBlockBitmap)
}

func (fs *FsSuper) InodeStart() buf.Bnum {
	return fs.BitmapInodeStart() + buf.Bnum(fs.NInodeBitmap)
}

func (fs *FsSuper) DataStart() buf.Bnum {
	return fs.InodeStart() + buf.Bnum(fs.nInodeBlk)
}

func (fs *FsSuper) Block2addr(blkno buf.Bnum) buf.Addr {
	return buf.MkAddr(blkno, 0, NBITBLOCK)
}

func (fs *FsSuper) NInode() Inum {
	return Inum(fs.nInodeBlk * INODEBLK)
}

func (fs *FsSuper) Inum2Addr(inum Inum) buf.Addr {
	return buf.MkAddr(fs.InodeStart()+buf.Bnum(uint64(inum)/INODEBLK),
		(uint64(inum)%INODEBLK)*INODESZ*8, INODESZ*8)
}

func (fs *FsSuper) DiskBlockSize(addr buf.Addr) bool {
	return addr.Sz == NBITBLOCK
}
