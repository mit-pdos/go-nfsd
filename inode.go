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

func roundupblk(n uint64) uint64 {
	return (n + disk.BlockSize - 1) / disk.BlockSize
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

func getInodeInumFree(txn *Txn, inum Inum) *Inode {
	ip := loadInode(txn, inum)
	if ip == nil {
		log.Printf("loadInode failed\n")
		return nil
	}
	ip.lock()
	return ip
}

func getInodeInum(txn *Txn, inum Inum) *Inode {
	ip := getInodeInumFree(txn, inum)
	if ip == nil {
		return nil
	}
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
	if ip.nlink == 0 {
		log.Printf("delete inode %d\n", ip.inum)
		if ip.kind != NF3FREE { // shrinker may put an FREE inode
			ip.resize(txn, 0)
			ip.freeInode(txn)
		}
		// if inode is allocated later a for new file (which
		// doesn't update the cache), this causes the inode to
		// be reloaded.
		ip.slot.obj = nil
	}
	txn.ic.delSlot(ip.inum)
}

//
// Freeing of a file, run in separate thread/transaction
//

// Returns blkno for indirect bn and newroot if root was allocated. If
// blkno is 0, failure.
func (ip *Inode) indbmap(txn *Txn, root uint64, level uint64, off uint64, grow bool) (uint64, uint64) {
	var newroot uint64 = 0
	var blkno uint64 = root

	if blkno == 0 { // no root?
		newroot = txn.AllocBlock()
		if newroot == 0 {
			return 0, 0
		}
		blkno = newroot
	}

	if level == 0 { // leaf?
		if root != 0 && grow { // old leaf?
			zeroBlock(txn, blkno)
		}
		return blkno, newroot
	}

	divisor := pow(level - 1)
	o := off / divisor
	bo := o * 8
	ind := off % divisor

	// log.Printf("indbmap: root %d off %d o %d\n", root, off, o)
	if root != 0 && off == 0 && grow { // old root from previous file?
		log.Printf("zero: blkno %d\n", blkno)
		zeroBlock(txn, blkno)
	}

	blk := txn.Read(blkno)
	nxtroot := machine.UInt64Get(blk[bo : bo+8])
	b, newroot1 := ip.indbmap(txn, nxtroot, level-1, ind, grow)
	if newroot1 != 0 {
		machine.UInt64Put(blk[bo:bo+8], newroot1)
		txn.WriteData(blkno, blk)
	}
	if b >= txn.fs.Size {
		panic("indbmap")
	}
	return b, newroot
}

// Lazily resize file. Bmap allocates/zeros blocks on demand.  Create
// a new thread to free blocks in a separate transaction.
func (ip *Inode) resize(txn *Txn, sz uint64) {
	log.Printf("resize %v to sz %d\n", ip, sz)
	if sz < ip.size {
		log.Printf("start shrink thread\n")
		txn.nfs.nthread = txn.nfs.nthread + 1
		go shrink(txn.nfs, ip.inum, ip.size)
	}
	ip.size = sz
}

// Map logical block number bn to a physical block number, allocating
// blocks if no block exists for bn. Reuse block from previous
// versions of this inode, but zero them.
func (ip *Inode) bmap(txn *Txn, bn uint64) (uint64, bool) {
	var alloc bool = false
	sz := roundupblk(ip.size)
	grow := bn > sz
	if bn < NDIRECT {
		if ip.blks[bn] != 0 && grow {
			zeroBlock(txn, ip.blks[bn])
		}
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
		off := bn - NDIRECT
		if off < NBLKBLK {
			blkno, newroot := ip.indbmap(txn, ip.blks[INDIRECT], 1, off, grow)
			if newroot != 0 {
				ip.blks[INDIRECT] = newroot
			}
			return blkno, newroot != 0
		} else {
			off = off - NBLKBLK
			blkno, newroot := ip.indbmap(txn, ip.blks[DINDIRECT], 2, off, grow)
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

	log.Printf("write %d %d ipsz %d\n", offset, count, ip.size)

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

func shrink(nfs *Nfs, inum Inum, oldsz uint64) {
	bn := roundupblk(oldsz)
	log.Printf("Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		txn := Begin(nfs)
		ip := getInodeInumFree(txn, inum)
		if ip == nil {
			panic("shrink")
		}
		if ip.size >= oldsz { // file has grown again, stop
			ok := txn.Commit([]*Inode{ip})
			if !ok {
				panic("shrink")
			}
			break
		}
		cursz := roundupblk(ip.size)
		log.Printf("shrink: bn %d cursz %d\n", bn, cursz)
		// 4: inode block, 2xbitmap block, indirect block, double indirect
		for bn > cursz && txn.numberDirty()+4 < txn.log.logSz {
			bn = bn - 1
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
		}
		ip.writeInode(txn)
		ok := txn.Commit([]*Inode{ip})
		if !ok {
			panic("shrink")
		}
		if bn <= cursz {
			break
		}
	}
	log.Printf("Shrinker: done shrinking %d to bn %d\n", inum, bn)
	nfs.mu.Lock()
	nfs.nthread = nfs.nthread - 1
	nfs.mu.Unlock()
	nfs.condShut.Signal()
}
