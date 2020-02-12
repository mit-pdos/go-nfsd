package super

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/bcache"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/util"
)

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
	nblockbitmap := (sz / common.NBITBLOCK) + 1
	var d disk.Disk
	if name != nil {
		util.DPrintf(0, "MkFsSuper: open file disk %s\n", *name)
		file, err := disk.NewFileDisk(*name, sz)
		if err != nil {
			panic("MkFsSuper: couldn't create disk image")
		}
		d = file
	} else {
		util.DPrintf(0, "MkFsSuper: create mem disk\n")
		d = disk.NewMemDisk(sz)
	}

	// use the disk with a buffer cache
	bc := bcache.MkBcache(d)

	return &FsSuper{
		Disk:         bc,
		Size:         sz,
		nLog:         common.LOGSIZE,
		NBlockBitmap: nblockbitmap,
		NInodeBitmap: common.NINODEBITMAP,
		nInodeBlk:    (common.NINODEBITMAP * common.NBITBLOCK * common.INODESZ) / disk.BlockSize,
		Maxaddr:      sz}
}

func (fs *FsSuper) MaxBnum() common.Bnum {
	return common.Bnum(fs.Maxaddr)
}

func (fs *FsSuper) BitmapBlockStart() common.Bnum {
	return common.Bnum(fs.nLog)
}

func (fs *FsSuper) BitmapInodeStart() common.Bnum {
	return fs.BitmapBlockStart() + common.Bnum(fs.NBlockBitmap)
}

func (fs *FsSuper) InodeStart() common.Bnum {
	return fs.BitmapInodeStart() + common.Bnum(fs.NInodeBitmap)
}

func (fs *FsSuper) DataStart() common.Bnum {
	return fs.InodeStart() + common.Bnum(fs.nInodeBlk)
}

func (fs *FsSuper) Block2addr(blkno common.Bnum) addr.Addr {
	return addr.MkAddr(blkno, 0, common.NBITBLOCK)
}

func (fs *FsSuper) NInode() common.Inum {
	return common.Inum(fs.nInodeBlk * common.INODEBLK)
}

func (fs *FsSuper) Inum2Addr(inum common.Inum) addr.Addr {
	return addr.MkAddr(fs.InodeStart()+common.Bnum(uint64(inum)/common.INODEBLK),
		(uint64(inum)%common.INODEBLK)*common.INODESZ*8, common.INODESZ*8)
}

func (fs *FsSuper) DiskBlockSize(addr addr.Addr) bool {
	return addr.Sz == common.NBITBLOCK
}
