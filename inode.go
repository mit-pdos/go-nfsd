package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

const NF3FREE Ftype3 = 0

const NDIRECT uint64 = 10

type Inode struct {
	// in-memory info:
	slot *Cslot
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
		slot:  nil,
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
		slot:  nil,
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

func (ip *Inode) encode(blk disk.Block) {
	enc := NewEnc(blk)
	enc.PutInt32(uint32(ip.kind))
	enc.PutInt32(ip.nlink)
	enc.PutInt(ip.gen)
	enc.PutInt(ip.size)
	enc.PutInts(ip.blks)
}

func decode(blk disk.Block, inum uint64) *Inode {
	ip := &Inode{}
	ip.slot = nil
	dec := NewDec(blk)
	ip.inum = inum
	ip.kind = Ftype3(dec.GetInt32())
	ip.nlink = dec.GetInt32()
	ip.gen = dec.GetInt()
	ip.size = dec.GetInt()
	ip.blks = dec.GetInts(NDIRECT)
	return ip
}

func loadInode(txn *Txn, inum Inum) *Inode {
	slot := txn.ic.lookupSlot(inum)
	if slot == nil {
		panic("loadInode")
	}
	ip := txn.fs.loadInode(txn, slot, inum)
	log.Printf("loadInode %v\n", ip)
	return ip
}

func getInodeInum(txn *Txn, inum Inum) *Inode {
	ip := loadInode(txn, inum)
	if ip == nil {
		log.Printf("loadInode failed\n")
		return nil
	}
	ip.lock()
	return ip
}

// Returns locked inode on success. This implicitly locks the inode
// block too.  If we put several inodes in a single inode block then
// we should lock the inode block explicitly (if the inode is already
// in cache).  (Or maybe delete the inode lock and always lock the
// block that contains the inode.)
func getInode(txn *Txn, fh3 Nfs_fh3) *Inode {
	fh := fh3.makeFh()
	ip := getInodeInum(txn, fh.ino)
	if ip.gen != fh.gen {
		log.Printf("wrong gen\n")
		ip.put(txn)
		ip.unlock()
		return nil
	}
	return ip
}

// To lock an inode, lock the reference in the cache slot
func (ip *Inode) lock() {
	//log.Printf("lock inum %d\n", ip.inum)
	ip.slot.lock()
}

func (ip *Inode) unlock() {
	//log.Printf("unlock inum %d\n", ip.inum)
	ip.slot.unlock()
}

func unlockInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.unlock()
	}
}

// Done with ip and remove inode if nlink and ref = 0. Must be run
// inside of a transaction since it may modify inode.
func (ip *Inode) put(txn *Txn) {
	log.Printf("put inode %d nlink %d\n", ip.inum, ip.nlink)
	last := txn.ic.delSlot(ip.inum)
	if last {
		if ip.nlink == 0 {
			log.Printf("delete inode %d\n", ip.inum)
			ip.resize(txn, 0)
			txn.fs.freeInode(txn, ip)
			ip.slot.obj = nil
		}
	}
}

func (ip *Inode) shrink(txn *Txn, sz uint64) bool {
	blocks := ip.size / disk.BlockSize
	if sz%disk.BlockSize != 0 {
		panic("shrink")
	}
	newsz := sz / disk.BlockSize
	for b := newsz; b < blocks; b++ {
		log.Printf("freeblock: %d\n", ip.blks[b])
		txn.fs.freeBlock(txn, ip.blks[b])
		ip.blks[b] = 0
	}
	ip.size = sz
	return true
}

func (ip *Inode) grow(txn *Txn, sz uint64) bool {
	n := sz / disk.BlockSize
	// XXX fix loop for goose
	for i := uint64(0); i < n; i++ {
		bn, ok := txn.fs.allocBlock(txn)
		log.Printf("allocblock: %d\n", bn)
		if !ok {
			return false
		}
		b := ip.size / disk.BlockSize
		ip.size = ip.size + disk.BlockSize
		ip.blks[b] = bn
		ok1 := txn.fs.writeInode(txn, ip)
		if !ok1 {
			panic("resize: writeInode failed")
		}
	}
	return true
}

func (ip *Inode) resize(txn *Txn, sz uint64) bool {
	var ok bool
	if sz < ip.size {
		ok = ip.shrink(txn, sz)
	} else {
		ok = ip.grow(txn, sz)
	}
	return ok
}

func (ip *Inode) bmap(txn *Txn, bn uint64) (uint64, bool) {
	var alloc bool = false
	if bn < NDIRECT {
		if ip.blks[bn] == 0 {
			blkno, ok := txn.fs.allocBlock(txn)
			log.Printf("allocblock: %d\n", blkno)
			if !ok {
				panic("bmap")
			}
			alloc = true
			ip.blks[bn] = blkno

		}
		return ip.blks[bn], alloc
	}
	return 0, alloc
}

func (ip *Inode) read(txn *Txn, offset uint64, count uint64) ([]byte, bool) {
	var n uint64 = uint64(0)

	if offset >= ip.size {
		return nil, true
	}
	if count >= offset+ip.size {
		count = ip.size - offset
	}
	data := make([]byte, count)
	for boff := offset / disk.BlockSize; n < count; boff++ {
		byteoff := offset % disk.BlockSize
		nbytes := disk.BlockSize - byteoff
		if count-n < nbytes {
			nbytes = count - n
		}
		blkno, alloc := ip.bmap(txn, boff)
		if alloc { // fill in a hole
			txn.fs.writeInode(txn, ip)
		}
		blk := txn.Read(blkno)
		// log.Printf("read off %d blkno %d %d %v..\n", n, blkno, nbytes, blk[0:32])
		for b := uint64(0); b < nbytes; b++ {
			data[n+b] = blk[byteoff+b]
		}
		n = n + nbytes
		offset = offset + nbytes
	}
	return data, false
}

func (ip *Inode) write(txn *Txn, offset uint64, count uint64, data []byte) (uint64, bool) {
	var cnt uint64 = uint64(0)
	var off uint64 = offset
	var ok bool = true
	var alloc bool = false
	n := count

	if offset+count > NDIRECT*disk.BlockSize {
		return 0, false
	}
	for boff := offset / disk.BlockSize; n > uint64(0); boff++ {
		blkno, new := ip.bmap(txn, boff)
		if new {
			alloc = true
		}
		blk := txn.Read(blkno)
		byteoff := offset % disk.BlockSize
		nbytes := disk.BlockSize - byteoff
		if n < nbytes {
			nbytes = n
		}
		for b := uint64(0); b < nbytes; b++ {
			blk[byteoff+b] = data[b]
		}
		ok := txn.WriteData(blkno, blk)
		if !ok {
			break
		}
		n = n - nbytes
		data = data[nbytes:]
		offset = offset + nbytes
		cnt = cnt + nbytes
	}
	if alloc || cnt > 0 {
		if off+cnt > ip.size {
			ip.size = off + cnt
		}
		ok := txn.fs.writeInode(txn, ip)
		if !ok {
			panic("write")
		}
	}
	return cnt, ok
}

func (ip *Inode) decLink(txn *Txn) bool {
	ip.nlink = ip.nlink - 1
	return txn.fs.writeInode(txn, ip)
}
