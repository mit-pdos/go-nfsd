package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

const NF3FREE Ftype3 = 0

const NDIRECT uint64 = 10

// XXX mu is unnecessary because transaction already has the inode block
// locked
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

const NULLINUM uint64 = 0
const ROOTINUM uint64 = 1

func mkNullInode() *Inode {
	return &Inode{
		mu:    new(sync.RWMutex),
		inum:  NULLINUM,
		kind:  NF3DIR,
		nlink: uint32(1),
		gen:   uint64(0),
		size:  uint64(0),
		blks:  make([]uint64, NDIRECT),
	}
}

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

func (ip *Inode) encode(blk disk.Block) disk.Block {
	enc := NewEnc(blk)
	enc.PutInt32(uint32(ip.kind))
	enc.PutInt32(ip.nlink)
	enc.PutInt(ip.gen)
	enc.PutInt(ip.size)
	enc.PutInts(ip.blks)
	return enc.Finish()
}

func decode(blk disk.Block, inum uint64) *Inode {
	ip := &Inode{}
	ip.mu = new(sync.RWMutex)
	dec := NewDec(blk)
	ip.inum = inum
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
	last := c.putObj(ip.inum, false)
	// XXX is this ok? is there a way ip could be resurrected
	// before we are done with it.
	if last && ip.nlink == 0 {
		// XXX truncate once we can create an inode with data
	}
}

func (ip *Inode) resize(fs *FsSuper, txn *Txn, sz uint64) bool {
	if sz < ip.size {
		panic("resize not implemented")
	}
	n := sz / disk.BlockSize
	// XXX fix loop for goose
	for i := uint64(0); i < n; i++ {
		bn, ok := fs.allocBlock(txn)
		log.Printf("allocblock: %d\n", bn)
		if !ok {
			return false
		}
		b := ip.size / disk.BlockSize
		ip.size = sz
		ip.blks[b] = bn
		ok1 := fs.writeInode(txn, ip)
		if !ok1 {
			panic("resize: writeInode failed")
		}
	}
	return true
}

const MaxNameLen = 4096 - 1 - 8

type DirEnt struct {
	Valid bool
	Name  string // max 4096-1-8=4087 bytes
	Inum  Inum
}

func encodeDirEnt(de *DirEnt, blk disk.Block) disk.Block {
	if len(de.Name) > MaxNameLen {
		panic("directory entry name too long")
	}
	enc := NewEnc(blk)
	enc.PutString(de.Name)
	enc.PutBool(de.Valid)
	enc.PutInt(de.Inum)
	return enc.Finish()
}

func decodeDirEnt(b disk.Block) *DirEnt {
	dec := NewDec(b)
	de := &DirEnt{}
	de.Name = dec.GetString()
	de.Valid = dec.GetBool()
	de.Inum = dec.GetInt()
	return de
}

func (ip *Inode) lookupLink(txn *Txn, name Filename3) uint64 {
	if ip.kind != NF3DIR {
		return 0
	}
	blocks := ip.size / disk.BlockSize
	for b := uint64(0); b < blocks; b++ {
		blk := (*txn).Read(ip.blks[b])
		de := decodeDirEnt(blk)
		if !de.Valid {
			continue
		}
		if de.Name == string(name) {
			return de.Inum
		}
	}
	return 0
}

func writeLink(blk disk.Block, txn *Txn, inum uint64, name Filename3, blkno uint64) bool {
	de := &DirEnt{Valid: true, Inum: inum, Name: string(name)}
	encodeDirEnt(de, blk)
	ok := (*txn).Write(blkno, blk)
	return ok
}

func (ip *Inode) addLink(fs *FsSuper, txn *Txn, inum uint64, name Filename3) bool {
	var freede *DirEnt

	if ip.kind != NF3DIR {
		return false
	}
	blocks := ip.size / disk.BlockSize
	for b := uint64(0); b < blocks; b++ {
		blk := (*txn).Read(ip.blks[b])
		de := decodeDirEnt(blk)
		if !de.Valid {
			writeLink(blk, txn, inum, name, ip.blks[b])
			freede = de
			break
		}
		continue
	}
	if freede != nil {
		return true
	}
	ok := ip.resize(fs, txn, ip.size+disk.BlockSize)
	if !ok {
		return false
	}
	blk := (*txn).Read(ip.blks[blocks])
	writeLink(blk, txn, inum, name, ip.blks[blocks])
	if !ok {
		panic("addLink")
	}
	return true
}
