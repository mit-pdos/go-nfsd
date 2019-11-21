package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
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

func (fs *FsSuper) getInode(txn *Txn, inum uint64) (bool, disk.Block) {
	if inum >= fs.ninodes {
		return false, nil
	}
	log.Printf("getInode %v %d\n", inum, fs.inode_start)
	blk := (*txn).Read(fs.inode_start + inum)
	return true, *blk
}

func (fs *FsSuper) loadInode(txn *Txn, co *Cobj, a uint64) *Inode {
	co.mu.Lock()
	if !co.valid {
		ok, blk := (*fs).getInode(txn, a)
		if !ok {
			return nil
		}
		i := decode(blk)
		co.obj = i
		co.valid = true
	}
	i := co.obj.(*Inode)
	co.mu.Unlock()
	return i
}

// XXX Check nlink
func (fs *FsSuper) putInode(txn *Txn, ip *Inode) {
}

// for mkfs
func (fs *FsSuper) putRootBlk(inum uint64, blk disk.Block) bool {
	if inum >= fs.ninodes {
		return false
	}
	log.Printf("write blk %d\n", fs.inode_start+inum+LOGEND)
	disk.Write(fs.inode_start+inum+LOGEND, blk)
	return true
}
