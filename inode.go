package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

const NF3FREE Ftype3 = 0

const NDIRECT uint64 = 10

type Inode struct {
	// in-memory info:
	mu    *sync.RWMutex
	inum  uint64
	valid bool
	ref   uint32
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
		valid: true,
		ref:   uint32(1),
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

func (ip *Inode) load(tx *Txn, fs *FsSuper) bool {
	ok, blk := (*fs).getInode(tx, ip.inum)
	if !ok {
		return false
	}
	dec := NewDec(blk)
	ip.kind = Ftype3(dec.GetInt32())
	ip.nlink = dec.GetInt32()
	ip.gen = dec.GetInt()
	ip.size = dec.GetInt()
	ip.blks = dec.GetInts(NDIRECT)
	ip.valid = true
	return true
}

func (ip *Inode) unlock() {
	ip.mu.Unlock()
}

func (ip *Inode) lock() {
	ip.mu.Lock()
}

const ICACHESZ uint64 = 10

type inodeCache struct {
	lock   *sync.RWMutex
	inodes []Inode
}

func mkInodeCache() *inodeCache {
	inodes := make([]Inode, ICACHESZ)
	n := uint64(len(inodes))
	for i := uint64(0); i < n; i++ {
		inodes[i].mu = new(sync.RWMutex)
	}
	return &inodeCache{
		lock:   new(sync.RWMutex),
		inodes: inodes,
	}
}

func (ic *inodeCache) getInode(inum uint64) *Inode {
	var ip *Inode
	var empty *Inode

	ic.lock.Lock()
	n := uint64(len(ic.inodes))
	for i := uint64(0); i < n; i++ {
		if ic.inodes[i].ref > 0 && ic.inodes[i].inum == inum {
			ip = &ic.inodes[i]
			break
		}
		if ic.inodes[i].ref == 0 && empty == nil {
			empty = &ic.inodes[i]
		}
		continue
	}
	if ip != nil {
		ip.ref = ip.ref + 1
		ic.lock.Unlock()
		return ip
	}
	if empty == nil {
		ic.lock.Unlock()
		return nil
	}
	ip = empty
	ip.inum = inum
	ip.valid = false
	ip.ref = 1
	ic.lock.Unlock()
	return ip
}

// XXX Check nlink
func (ic *inodeCache) putInode(ip *Inode) {
	ic.lock.Lock()
	ip.ref = ip.ref - 1
	ic.lock.Unlock()
}

func allocInode(tx *Txn, kind Ftype3) *Inode {
	return nil
}
