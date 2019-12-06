package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"fmt"
	"log"
)

const NF3FREE Ftype3 = 0

const (
	NBLKINO   uint64 = 10 // # blk in an inode's blks array
	NDIRECT   uint64 = NBLKINO - 2
	INDIRECT  uint64 = NBLKINO - 2
	DINDIRECT uint64 = NBLKINO - 1
	NBLKBLK   uint64 = disk.BlockSize / 8 // # blkno per block
	NINDLEVEL uint64 = 2                  // # levels of indirection
)

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
		blks:  make([]uint64, NBLKINO),
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
		blks:  make([]uint64, NBLKINO),
	}
}

func (ip *Inode) String() string {
	return fmt.Sprintf("# %d k %d n %d g %d sz %d %v", ip.inum, ip.kind, ip.nlink, ip.gen, ip.size, ip.blks)
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
	ip.blks = dec.GetInts(NBLKINO)
	return ip
}

func pow(level uint64) uint64 {
	if level == 0 {
		return 1
	}
	var p uint64 = NBLKBLK
	for i := uint64(1); i < level; i++ {
		p = p * p
	}
	return p
}

func MaxFileSize() uint64 {
	maxblks := pow(NINDLEVEL)
	return (NDIRECT + maxblks) * disk.BlockSize
}

func getInodeInum(txn *Txn, inum Inum) *Inode {
	ip := loadInode(txn, inum)
	if ip == nil {
		log.Printf("loadInode failed\n")
		return nil
	}
	ip.lock()
	if ip.kind == NF3FREE {
		log.Printf("inode is free\n")
		ip.put(txn)
		ip.unlock()
		return nil
	}
	if ip.nlink == 0 {
		panic("getInodeInum")
	}
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
	if ip == nil {
		return nil
	}
	if ip.gen != fh.gen {
		log.Printf("non existent ip or wrong gen\n")
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
	log.Printf("readInode %v\n", i)
	return i
}

func (ip *Inode) writeInode(txn *Txn) {
	blk := txn.readInodeBlock(ip.inum)
	log.Printf("writeInode %v\n", ip)
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

// Done with ip and remove inode if nlink = 0. Must be run inside of a
// transaction since it may modify inode, and assumes caller has lock
// on inode.
func (ip *Inode) put(txn *Txn) {
	log.Printf("put inode %d nlink %d\n", ip.inum, ip.nlink)
	if ip.nlink == 0 && ip.kind != NF3FREE {
		log.Printf("delete inode %d\n", ip.inum)
		ip.resize(txn, 0)
		ip.freeInode(txn)
		// if inode is allocated later a for new file (which
		// doesn't update the cache), this causes the inode to
		// be reloaded.
		ip.slot.obj = nil
	}
	txn.ic.delSlot(ip.inum)
}

// Returns blkno for indirect bn and newroot if root was allocated. If
// blkno is 0, failure.
func (ip *Inode) indbmap(txn *Txn, root uint64, level uint64, bn uint64) (uint64, uint64) {
	var newroot uint64 = 0
	var blkno uint64 = root
	if blkno == 0 {
		newroot = txn.AllocBlock()
		if newroot == 0 {
			return 0, 0
		}
		blkno = newroot
	}
	if level == 0 {
		return blkno, newroot
	}
	divisor := pow(level - 1)
	off := bn / divisor
	ind := bn % divisor
	boff := off * 8
	blk := txn.Read(blkno)
	nxtroot := machine.UInt64Get(blk[boff : boff+8])
	b, newroot1 := ip.indbmap(txn, nxtroot, level-1, ind)
	if newroot1 != 0 {
		machine.UInt64Put(blk[boff:boff+8], newroot1)
		txn.WriteData(blkno, blk)
	}
	if b >= txn.fs.Size {
		panic("indbmap")
	}
	return b, newroot
}

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *Inode) indshrink(txn *Txn, root uint64, level uint64, bn uint64) uint64 {
	if level == 0 {
		return root
	}
	divisor := pow(level - 1)
	off := (bn / divisor)
	ind := bn % divisor
	boff := off * 8
	blk := txn.Read(root)
	nxtroot := machine.UInt64Get(blk[boff : boff+8])
	if nxtroot != 0 {
		freeroot := ip.indshrink(txn, nxtroot, level-1, ind)
		if freeroot != 0 {
			machine.UInt64Put(blk[boff:boff+8], 0)
			txn.WriteData(root, blk)
			txn.FreeBlock(freeroot)
		}
	}
	if off == 0 && ind == 0 {
		return root
	} else {
		return 0
	}
}

func (ip *Inode) shrink(txn *Txn, sz uint64) bool {
	var bn uint64
	oldsz := ip.size / disk.BlockSize
	if sz%disk.BlockSize != 0 {
		panic("shrink")
	}
	newsz := sz / disk.BlockSize
	bn = oldsz - 1
	for oldsz != 0 {
		log.Printf("freeblock: %d\n", bn)
		if bn < NDIRECT {
			txn.FreeBlock(ip.blks[bn])
			ip.blks[bn] = 0
		} else {
			off := bn - NDIRECT
			if off < NBLKBLK {
				freeroot := ip.indshrink(txn, ip.blks[INDIRECT], 1, off)
				if freeroot != 0 {
					txn.FreeBlock(ip.blks[INDIRECT])
					ip.blks[INDIRECT] = 0
				}
			} else {
				off = off - NBLKBLK
				freeroot := ip.indshrink(txn, ip.blks[DINDIRECT], 2, off)
				if freeroot != 0 {
					txn.FreeBlock(ip.blks[DINDIRECT])
					ip.blks[DINDIRECT] = 0
				}
			}
		}
		if bn == newsz {
			break
		}
		bn = bn - 1
		continue
	}
	ip.size = sz
	return true
}

// Lazily grow file. Bmap allocates blocks on demand
func (ip *Inode) grow(txn *Txn, sz uint64) bool {
	if sz%disk.BlockSize != 0 {
		panic("grow")
	}
	ip.size = sz
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
			blkno := txn.AllocBlock()
			if blkno == 0 {
				return 0, false
			}
			alloc = true
			ip.blks[bn] = blkno

		}
		return ip.blks[bn], alloc
	} else {
		bn = bn - NDIRECT
		if bn < NBLKBLK {
			blkno, newroot := ip.indbmap(txn, ip.blks[INDIRECT],
				1, bn)
			if newroot != 0 {
				ip.blks[INDIRECT] = newroot
			}
			return blkno, newroot != 0
		} else {
			bn = bn - NBLKBLK
			blkno, newroot := ip.indbmap(txn, ip.blks[DINDIRECT],
				2, bn)
			if newroot != 0 {
				ip.blks[DINDIRECT] = newroot
			}
			return blkno, newroot != 0
		}
	}
	return 0, false
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

// Returns number of bytes written and error
func (ip *Inode) write(txn *Txn, offset uint64, count uint64, data []byte) (uint64, bool) {
	var cnt uint64 = uint64(0)
	var off uint64 = offset
	var ok bool = true
	var alloc bool = false
	n := count

	if offset+count > MaxFileSize() {
		return 0, false
	}
	for boff := offset / disk.BlockSize; n > uint64(0); boff++ {
		blkno, new := ip.bmap(txn, boff)
		if blkno == 0 {
			ok = false
			break
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
