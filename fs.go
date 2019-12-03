package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

type FsSuper struct {
	NLog    uint64 // including commit block
	NBitmap uint64
	NInode  uint64
	MaxAddr uint64
}

func mkFsSuper() *FsSuper {
	sz := uint64(10 * 10000)
	nbitmap := (sz / disk.BlockSize) + 1
	disk.Init(disk.NewMemDisk(sz))
	return &FsSuper{NLog: LOGSIZE, NBitmap: nbitmap, NInode: 2000, MaxAddr: sz}
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

func (fs *FsSuper) allocBlock(txn *Txn) (uint64, bool) {
	var found bool = false
	var bit uint64 = 0

	for i := uint64(0); i < fs.NBitmap; i++ {
		blkno := fs.bitmapStart() + i
		blk := txn.Read(blkno)
		bit, found = findAndMark(blk)
		if !found {
			txn.ReleaseBlock(blkno)
			continue
		}
		ok := txn.Write(blkno, blk)
		if !ok {
			panic("allocBlock")
		}
		bit = i*disk.BlockSize + bit
		break
	}
	return bit, found
}

func (fs *FsSuper) freeBlock(txn *Txn, bn uint64) {
	i := bn / disk.BlockSize
	if i >= fs.NBitmap {
		panic("freeBlock")
	}
	blkno := fs.bitmapStart() + i
	blk := txn.Read(blkno)
	freeBit(blk, bn%disk.BlockSize)
	ok1 := txn.Write(blkno, blk)
	if !ok1 {
		panic("freeBlock")
	}
}

func (fs *FsSuper) readInodeBlock(txn *Txn, inum uint64) (disk.Block, bool) {
	if inum >= fs.NInode {
		return nil, false
	}
	blk := txn.Read(fs.inodeStart() + inum)
	return blk, true
}

func (fs *FsSuper) readInode(txn *Txn, inum uint64) (*Inode, bool) {
	if inum >= fs.NInode {
		return nil, false
	}
	blk, ok := fs.readInodeBlock(txn, inum)
	i := decode(blk, inum)
	log.Printf("readInode %v %v\n", inum, i)
	return i, ok
}

func (fs *FsSuper) writeInodeBlock(txn *Txn, inum uint64, blk disk.Block) bool {
	if inum >= fs.NInode {
		return false
	}
	ok := txn.Write(fs.inodeStart()+inum, blk)
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

func (fs *FsSuper) releaseInodeBlock(txn *Txn, inum uint64) bool {
	if inum >= fs.NInode {
		return false
	}
	txn.ReleaseBlock(fs.inodeStart() + inum)
	return true
}

func (fs *FsSuper) loadInode(txn *Txn, slot *Cslot, a uint64) *Inode {
	slot.lock()
	if slot.obj == nil {
		i, ok := fs.readInode(txn, a)
		if !ok {
			return nil
		}
		i.slot = slot
		slot.obj = i
	}
	i := slot.obj.(*Inode)
	slot.unlock()
	return i
}

func (fs *FsSuper) allocInode(txn *Txn, kind Ftype3) Inum {
	var inode *Inode
	for inum := uint64(1); inum < fs.NInode; inum++ {
		i, ok := fs.readInode(txn, inum)
		if !ok {
			break
		}
		if i.kind == NF3FREE {
			log.Printf("allocInode: allocate inode %d\n", inum)
			inode = i
			inode.inum = inum
			inode.kind = kind
			inode.nlink = 1
			inode.gen = inode.gen + 1
			break
		}
		fs.releaseInodeBlock(txn, inum)
		continue
	}
	if inode == nil {
		return 0
	}
	_ = fs.writeInode(txn, inode)
	return inode.inum
}

func (fs *FsSuper) freeInode(txn *Txn, i *Inode) bool {
	i.kind = NF3FREE
	i.gen = i.gen + 1
	return fs.writeInode(txn, i)
}

func (fs *FsSuper) freeInum(txn *Txn, inum Inum) bool {
	i, ok := fs.readInode(txn, inum)
	if !ok {
		panic("freeInode")
	}
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	return fs.freeInode(txn, i)
}

//
// for mkfs
//

func (fs *FsSuper) markAlloc(n uint64, m uint64) {
	log.Printf("markAlloc: [0, %d) and [%d,%d)\n", n, m, fs.NBitmap*disk.BlockSize)
	if n >= disk.BlockSize || m >= disk.BlockSize*fs.NBitmap || m < disk.BlockSize {
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
	blkno := m/disk.BlockSize + fs.bitmapStart()
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
