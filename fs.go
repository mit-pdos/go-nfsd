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
	return &FsSuper{bitmap: 1, ninodes: 10, inode_start: 2}
}

func (fs *FsSuper) readInodeBlock(txn *Txn, inum uint64) (bool, disk.Block) {
	if inum >= fs.ninodes {
		return false, nil
	}
	blk := (*txn).Read(fs.inode_start + inum)
	return true, *blk
}

func (fs *FsSuper) readInode(txn *Txn, inum uint64) (bool, *Inode) {
	if inum >= fs.ninodes {
		return false, nil
	}
	blk := (*txn).Read(fs.inode_start + inum)
	i := decode(blk)
	log.Printf("readInode %v %v\n", inum, i)
	return true, i
}

func (fs *FsSuper) writeInodeBlock(txn *Txn, inum uint64, blk *disk.Block) bool {
	if inum >= fs.ninodes {
		return false
	}
	ok := (*txn).Write(fs.inode_start+inum, blk)
	if !ok {
		panic("writeInodeBlock")
	}
	return true
}

func (fs *FsSuper) writeInode(txn *Txn, inode *Inode) bool {
	blk := inode.encode()
	log.Printf("writeInode %v\n", inode.inum)
	return fs.writeInodeBlock(txn, inode.inum, blk)
}

func (fs *FsSuper) loadInode(txn *Txn, co *Cobj, a uint64) *Inode {
	co.mu.Lock()
	if !co.valid {
		ok, i := (*fs).readInode(txn, a)
		if !ok {
			return nil
		}
		co.obj = i
		co.valid = true
	}
	i := co.obj.(*Inode)
	co.mu.Unlock()
	return i
}

func (fs *FsSuper) allocInode(txn *Txn, kind Ftype3) *Inode {
	var inode *Inode
	for inum := uint64(1); inum < fs.ninodes; inum++ {
		ok, i := fs.readInode(txn, inum)
		if !ok {
			break
		}
		log.Printf("allocInode: %d\n", inum)
		if i.kind == NF3FREE {
			inode = i
			inode.inum = inum
			break
		}
		// XXX release inode block from txn
		continue
	}
	if inode == nil {
		return nil
	}
	inode.kind = kind
	_ = fs.writeInode(txn, inode)
	return inode
}

// for mkfs
func (fs *FsSuper) putRootBlk(inum uint64, blk *disk.Block) bool {
	if inum >= fs.ninodes {
		return false
	}
	log.Printf("write blk %d\n", fs.inode_start+inum+LOGEND)
	disk.Write(fs.inode_start+inum+LOGEND, *blk)
	return true
}
