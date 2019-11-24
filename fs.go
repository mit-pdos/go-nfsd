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

func (fs *FsSuper) initFs() {
	nulli := mkNullInode()
	nullblk := make(disk.Block, disk.BlockSize)
	nulli.encode(nullblk)
	fs.putBlkDirect(NULLINUM, nullblk)
	root := mkRootInode()
	rootblk := make(disk.Block, disk.BlockSize)
	root.encode(rootblk)
	fs.putBlkDirect(ROOTINUM, rootblk)
	fs.markAlloc(fs.inodeStart() + fs.nInode)
}

// Find a free bit in blk and toggle it
func findAndMark(blk disk.Block) (uint64, bool) {
	for byte := uint64(0); byte < disk.BlockSize; byte++ {
		byteVal := blk[byte]
		if byteVal == 0xff {
			continue
		}
		for bit := uint64(0); bit < 8; bit++ {
			if byteVal&(1<<bit) == 0 {
				blk[byte] |= 1 << bit
				off := bit + 8*byte
				return off, true
			}
		}
	}
	return 0, false
}

// Toggle bit bn in blk
func freeBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] = blk[byte] & ^(1 << bit)
}

// XXX support several bitmap blocks
func (fs *FsSuper) allocBlock(txn *Txn) (uint64, bool) {
	blk := (*txn).Read(fs.bitmapStart())
	bit, ok := findAndMark(blk)
	if !ok {
		return 0, false
	}
	ok1 := (*txn).Write(fs.bitmapStart(), blk)
	if !ok1 {
		panic("allocBlock")
	}
	return bit, true
}

func (fs *FsSuper) freeBlock(txn *Txn, bn uint64) {
	blk := (*txn).Read(fs.bitmapStart())
	freeBit(blk, bn)
	ok1 := (*txn).Write(fs.bitmapStart(), blk)
	if !ok1 {
		panic("freeBlock")
	}
}

func (fs *FsSuper) readInodeBlock(txn *Txn, inum uint64) (disk.Block, bool) {
	if inum >= fs.nInode {
		return nil, false
	}
	blk := (*txn).Read(fs.inodeStart() + inum)
	return blk, true
}

func (fs *FsSuper) readInode(txn *Txn, inum uint64) (*Inode, bool) {
	if inum >= fs.nInode {
		return nil, false
	}
	blk, ok := fs.readInodeBlock(txn, inum)
	i := decode(blk, inum)
	log.Printf("readInode %v %v\n", inum, i)
	return i, ok
}

func (fs *FsSuper) writeInodeBlock(txn *Txn, inum uint64, blk disk.Block) bool {
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
	blk, ok := fs.readInodeBlock(txn, inode.inum)
	if !ok {
		return false
	}
	log.Printf("writeInode %d %v\n", inode.inum, inode)
	inode.encode(blk)
	return fs.writeInodeBlock(txn, inode.inum, blk)
}

func (fs *FsSuper) loadInode(txn *Txn, slot *Cslot, a uint64) *Inode {
	slot.mu.Lock()
	if slot.obj == nil {
		i, ok := (*fs).readInode(txn, a)
		if !ok {
			return nil
		}
		slot.obj = i
	}
	i := slot.obj.(*Inode)
	slot.mu.Unlock()
	return i
}

func (fs *FsSuper) allocInode(txn *Txn, kind Ftype3) Inum {
	var inode *Inode
	for inum := uint64(1); inum < fs.nInode; inum++ {
		i, ok := fs.readInode(txn, inum)
		if !ok {
			break
		}
		if i.kind == NF3FREE {
			log.Printf("allocInode: allocate inode %d\n", inum)
			inode = i
			inode.inum = inum
			inode.kind = kind
			break
		}
		// XXX release inode block from txn
		continue
	}
	if inode == nil {
		return 0
	}
	_ = fs.writeInode(txn, inode)
	return inode.inum
}

func (fs *FsSuper) freeInode(txn *Txn, inum Inum) {
	i, ok := fs.readInode(txn, inum)
	if !ok {
		panic("freeInode")
	}
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	i.kind = NF3FREE
	_ = fs.writeInode(txn, i)
}

// for mkfs

// XXX mark bn > maximum size of disk has allocated
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

func (fs *FsSuper) putBlkDirect(inum uint64, blk disk.Block) bool {
	if inum >= fs.nInode {
		return false
	}
	log.Printf("write blk direct %d\n", fs.inodeStart()+inum)
	disk.Write(fs.inodeStart()+inum, blk)
	return true
}
