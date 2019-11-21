package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

const NF3FREE Ftype3 = 0

const NDIRECT uint64 = 10

type Inode struct {
	// in-memory info:
	mu   *sync.RWMutex
	inum uint64
	// the on-disk inode:
	kind  Ftype3
	nlink uint32
	gen   uint64
	size  uint64
	blks  []uint64
}

type Inum = uint64

const ROOTINUM uint64 = 0

func mkRootInode() *Inode {
	return &Inode{
		mu:    new(sync.RWMutex),
		inum:  ROOTINUM,
		kind:  NF3DIR,
		nlink: uint32(1),
		gen:   uint64(0),
		size:  uint64(0),
		blks:  make([]uint64, NDIRECT),
	}
}

func (ip *Inode) mkFattr() Fattr3 {
	return Fattr3{
		Ftype:  ip.kind,
		Mode:   0777,
		Nlink:  1,
		Uid:    Uid3(0),
		Gid:    Gid3(0),
		Size:   Size3(ip.size),
		Used:   Size3(ip.size),
		Rdev:   Specdata3{Specdata1: Uint32(0), Specdata2: Uint32(0)},
		Fsid:   Uint64(0),
		Fileid: Fileid3(0),
		Atime:  Nfstime3{Seconds: Uint32(0), Nseconds: Uint32(0)},
		Mtime:  Nfstime3{Seconds: Uint32(0), Nseconds: Uint32(0)},
		Ctime:  Nfstime3{Seconds: Uint32(0), Nseconds: Uint32(0)},
	}
}

func (ip *Inode) encode() disk.Block {
	enc := NewEnc()
	enc.PutInt32(uint32(ip.kind))
	enc.PutInt32(ip.nlink)
	enc.PutInt(ip.gen)
	enc.PutInt(ip.size)
	enc.PutInts(ip.blks)
	return enc.Finish()
}

func decode(blk disk.Block) *Inode {
	ip := &Inode{}
	ip.mu = new(sync.RWMutex)
	dec := NewDec(blk)
	ip.kind = Ftype3(dec.GetInt32())
	ip.nlink = dec.GetInt32()
	ip.gen = dec.GetInt()
	ip.size = dec.GetInt()
	ip.blks = dec.GetInts(NDIRECT)
	return ip
}

func (ip *Inode) unlock() {
	ip.mu.Unlock()
}

func (ip *Inode) lock() {
	ip.mu.Lock()
}

func (ip *Inode) putInode(c *Cache, txn *Txn) {
	log.Printf("put inode %d\n", ip.inum)
	last := c.putObj(ip.inum)
	// XXX is this ok? is there a way ip could be resurrected
	// before we are done with it.
	if last && ip.nlink == 0 {
		// XXX truncate once we can create an inode with data
	}
}

func allocInode(tx *Txn, kind Ftype3) *Inode {
	return nil
}
