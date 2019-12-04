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

func getInodeInum(txn *Txn, inum Inum) *Inode {
	ip := loadInode(txn, inum)
	if ip == nil {
		log.Printf("loadInode failed\n")
		return nil
	}
	ip.lock()
	return ip
}

func loadInode(txn *Txn, inum Inum) *Inode {
	if inum >= txn.fs.NInode {
		return nil
	}
	slot := txn.ic.lookupSlot(inum)
	if slot == nil {
		panic("loadInode")
	}
	ip := loadInodeSlot(txn, slot, inum)
	log.Printf("loadInode %v\n", ip)
	return ip
}

func loadInodeSlot(txn *Txn, slot *Cslot, inum Inum) *Inode {
	slot.lock()
	if slot.obj == nil {
		i := readInode(txn, inum)
		i.slot = slot
		slot.obj = i
	}
	i := slot.obj.(*Inode)
	slot.unlock()
	return i
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

func readInode(txn *Txn, inum uint64) *Inode {
	blk := txn.readInodeBlock(inum)
	i := decode(blk, inum)
	log.Printf("readInode %v %v\n", inum, i)
	return i
}

func (ip *Inode) writeInode(txn *Txn) {
	blk := txn.readInodeBlock(ip.inum)
	log.Printf("writeInode %d %v\n", ip.inum, ip)
	ip.encode(blk)
	txn.writeInodeBlock(ip.inum, blk)
}

func allocInode(txn *Txn, kind Ftype3) Inum {
	var inode *Inode
	for inum := uint64(1); inum < txn.fs.NInode; inum++ {
		i := readInode(txn, inum)
		if i.kind == NF3FREE {
			log.Printf("allocInode: allocate inode %d\n", inum)
			inode = i
			inode.inum = inum
			inode.kind = kind
			inode.nlink = 1
			inode.gen = inode.gen + 1
			break
		}
		// Remove this unused block from txn
		txn.releaseInodeBlock(inum)
		continue
	}
	if inode == nil {
		return 0
	}
	inode.writeInode(txn)
	return inode.inum
}

func (ip *Inode) freeInode(txn *Txn) {
	ip.kind = NF3FREE
	ip.gen = ip.gen + 1
	ip.writeInode(txn)
}

func freeInum(txn *Txn, inum Inum) {
	i := readInode(txn, inum)
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(txn)
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
			ip.freeInode(txn)
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
		bn := txn.fs.allocBlock(txn)
		log.Printf("allocblock: %d\n", bn)
		if bn == 0 {
			return false
		}
		b := ip.size / disk.BlockSize
		ip.size = ip.size + disk.BlockSize
		ip.blks[b] = bn
		ip.writeInode(txn)
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
			blkno := txn.fs.allocBlock(txn)
			log.Printf("allocblock: %d\n", blkno)
			if blkno == 0 {
				return 0, false
			}
			alloc = true
			ip.blks[bn] = blkno

		}
		return ip.blks[bn], alloc
	}
	return 0, alloc
}

// Returns number of bytes read and eof
func (ip *Inode) read(txn *Txn, offset uint64, count uint64) ([]byte, bool) {
	var n uint64 = uint64(0)

	if offset >= ip.size {
		return nil, true
	}
	if count >= offset+ip.size {
		count = ip.size - offset
	}
	data := make([]byte, 0)
	for boff := offset / disk.BlockSize; n < count; boff++ {
		byteoff := offset % disk.BlockSize
		nbytes := disk.BlockSize - byteoff
		if count-n < nbytes {
			nbytes = count - n
		}
		blkno, alloc := ip.bmap(txn, boff)
		if blkno == 0 {
			return data, false
		}
		if alloc { // fill in a hole
			ip.writeInode(txn)
		}
		blk := txn.Read(blkno)
		// log.Printf("read off %d blkno %d %d %v..\n", n, blkno, nbytes, blk[0:32])
		for b := uint64(0); b < nbytes; b++ {
			data = append(data, blk[byteoff+b])
		}
		n = n + nbytes
		offset = offset + nbytes
	}
	return data, false
}

// Returns number of bytes written and eof
// XXX return error on bmap failure?
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
		if blkno == 0 {
			return cnt, false
		}
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
		txn.WriteData(blkno, blk)
		n = n - nbytes
		data = data[nbytes:]
		offset = offset + nbytes
		cnt = cnt + nbytes
	}
	if alloc || cnt > 0 {
		if off+cnt > ip.size {
			ip.size = off + cnt
		}
		ip.writeInode(txn)
	}
	return cnt, ok
}

func (ip *Inode) decLink(txn *Txn) {
	ip.nlink = ip.nlink - 1
	ip.writeInode(txn)
}
