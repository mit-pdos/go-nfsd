package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

type FsSuper struct {
	nLog    uint64 // including commit block
	nBitmap uint64
	nInode  uint64
	maxAddr uint64
}

func mkFsSuper() *FsSuper {
	sz := uint64(10 * 1000)
	disk.Init(disk.NewMemDisk(sz))
	return &FsSuper{nLog: 60, nBitmap: 1, nInode: 10, maxAddr: sz}
}

func (fs *FsSuper) bitmapStart() uint64 {
	return fs.nLog
}

func (fs *FsSuper) inodeStart() uint64 {
	return fs.nLog + fs.nBitmap
}

func (fs *FsSuper) dataStart() uint64 {
	return fs.nLog + fs.nBitmap + fs.nInode
}

func findMarkAlloc(blk *disk.Block) (uint64, bool) {
	for byte := uint64(0); byte < disk.BlockSize; byte++ {
		byteVal := (*blk)[byte]
		if byteVal == 0xff {
			continue
		}
		for bit := uint64(0); bit < 8; bit++ {
			if byteVal&(1<<bit) == 0 {
				(*blk)[byte] |= 1 << bit
				off := bit + 8*byte
				return off, true
			}
		}
	}
	return 0, false
}

func (fs *FsSuper) allocBlock(txn *Txn) (uint64, bool) {
	blk := (*txn).Read(fs.bitmapStart())
	bit, ok := findMarkAlloc(blk)
	if !ok {
		return 0, false
	}
	ok1 := (*txn).Write(fs.bitmapStart(), blk)
	if !ok1 {
		panic("allocBlock")
	}
	return bit, true
}

func (fs *FsSuper) readInodeBlock(txn *Txn, inum uint64) (bool, disk.Block) {
	if inum >= fs.nInode {
		return false, nil
	}
	blk := (*txn).Read(fs.inodeStart() + inum)
	return true, *blk
}

func (fs *FsSuper) readInode(txn *Txn, inum uint64) (bool, *Inode) {
	if inum >= fs.nInode {
		return false, nil
	}
	blk := (*txn).Read(fs.inodeStart() + inum)
	i := decode(blk)
	log.Printf("readInode %v %v\n", inum, i)
	return true, i
}

func (fs *FsSuper) writeInodeBlock(txn *Txn, inum uint64, blk *disk.Block) bool {
	if inum >= fs.nInode {
		return false
	}
	ok := (*txn).Write(fs.inodeStart()+inum, blk)
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

func (fs *FsSuper) allocInode(txn *Txn, kind Ftype3) uint64 {
	var inode *Inode
	for inum := uint64(1); inum < fs.nInode; inum++ {
		ok, i := fs.readInode(txn, inum)
		if !ok {
			break
		}
		log.Printf("allocInode: allocate inode %d\n", inum)
		if i.kind == NF3FREE {
			inode = i
			inode.inum = inum
			break
		}
		// XXX release inode block from txn
		continue
	}
	if inode == nil {
		return 0
	}
	inode.kind = kind
	_ = fs.writeInode(txn, inode)
	return inode.inum
}

// for mkfs

// XXX deal with maximum size of disk
func (fs *FsSuper) markAlloc(n uint64) {
	log.Printf("markAlloc: %d\n", n)
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < n; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] |= 1 << bit
	}
	disk.Write(fs.bitmapStart(), blk)
}

func (fs *FsSuper) putRootBlk(inum uint64, blk *disk.Block) bool {
	if inum >= fs.nInode {
		return false
	}
	log.Printf("write blk %d\n", fs.inodeStart()+inum)
	disk.Write(fs.inodeStart()+inum, *blk)
	return true
}
