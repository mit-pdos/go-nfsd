package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"
)

type FsSuper struct {
	bitmap      uint64
	ninodes     uint64
	inode_start uint64
}

func mkFsSuper() *FsSuper {
	disk.Init(disk.NewMemDisk(10 * 1000))
	return &FsSuper{bitmap: 1, ninodes: 1, inode_start: 2}
}

func (fs *FsSuper) getInode(tx *Txn, inum uint64) (bool, disk.Block) {
	if inum >= fs.ninodes {
		return false, nil
	}
	blk := (*tx).Read(fs.inode_start + inum)
	return true, blk
}

// for mkfs
func (fs *FsSuper) putRootBlk(inum uint64, blk disk.Block) bool {
	if inum >= fs.ninodes {
		return false
	}
	disk.Write(fs.inode_start+inum+LOGEND, blk)
	return true
}
