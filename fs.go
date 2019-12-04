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

// Zero indicates failure
func (fs *FsSuper) allocBlock(txn *Txn) uint64 {
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
		txn.Write(blkno, blk)
		bit = i*disk.BlockSize + bit
		break
	}
	return bit
}

func (fs *FsSuper) freeBlock(txn *Txn, bn uint64) {
	i := bn / disk.BlockSize
	if i >= fs.NBitmap {
		panic("freeBlock")
	}
	blkno := fs.bitmapStart() + i
	blk := txn.Read(blkno)
	freeBit(blk, bn%disk.BlockSize)
	txn.Write(blkno, blk)
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
