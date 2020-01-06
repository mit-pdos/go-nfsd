package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"fmt"
)

const NF3FREE Ftype3 = 0

const (
	NBLKINO   uint64 = 5 // # blk in an inode's blks array
	NDIRECT   uint64 = NBLKINO - 2
	INDIRECT  uint64 = NBLKINO - 2
	DINDIRECT uint64 = NBLKINO - 1
	NBLKBLK   uint64 = disk.BlockSize / 8 // # blkno per block
	NINDLEVEL uint64 = 2                  // # levels of indirection
	INODESZ   uint64 = 64                 // on-disk size
)

type inum uint64

const NULLINUM inum = 0
const ROOTINUM inum = 1

type inode struct {
	// in-memory info:
	inum inum

	// the on-disk inode:
	kind  Ftype3
	nlink uint32
	gen   uint64
	size  uint64
	blks  []uint64
}

func mkNullInode() *inode {
	return &inode{
		inum:  NULLINUM,
		kind:  NF3DIR,
		nlink: uint32(1),
		gen:   uint64(0),
		size:  uint64(0),
		blks:  make([]uint64, NBLKINO),
	}
}

func mkRootInode() *inode {
	return &inode{
		inum:  ROOTINUM,
		kind:  NF3DIR,
		nlink: uint32(1),
		gen:   uint64(0),
		size:  uint64(0),
		blks:  make([]uint64, NBLKINO),
	}
}

func (ip *inode) String() string {
	return fmt.Sprintf("# %d k %d n %d g %d sz %d %v", ip.inum, ip.kind, ip.nlink, ip.gen, ip.size, ip.blks)
}

func (ip *inode) mkFattr() Fattr3 {
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

func (ip *inode) encode(buf *buf) {
	enc := newEnc(buf.blk)
	enc.putInt32(uint32(ip.kind))
	enc.putInt32(ip.nlink)
	enc.putInt(ip.gen)
	enc.putInt(ip.size)
	enc.putInts(ip.blks)
}

func decode(buf *buf, inum inum) *inode {
	ip := &inode{
		inum:  0,
		kind:  0,
		nlink: 0,
		gen:   0,
		size:  0,
		blks:  nil,
	}
	dec := newDec(buf.blk)
	ip.inum = inum
	ip.kind = Ftype3(dec.getInt32())
	ip.nlink = dec.getInt32()
	ip.gen = dec.getInt()
	ip.size = dec.getInt()
	ip.blks = dec.getInts(NBLKINO)
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

func maxFileSize() uint64 {
	maxblks := pow(NINDLEVEL)
	return (NDIRECT + maxblks) * disk.BlockSize
}

func getInodeLocked(txn *txn, inum inum) *inode {
	addr := txn.fs.inum2addr(inum)
	buf := txn.readBufLocked(addr, INODE)
	i := decode(buf, inum)
	dPrintf(5, "getInodeLocked %v\n", i)
	return i
}

func getInodeInumFree(txn *txn, inum inum) *inode {
	ip := getInodeLocked(txn, inum)
	return ip
}

func getInodeInum(txn *txn, inum inum) *inode {
	ip := getInodeInumFree(txn, inum)
	if ip == nil {
		return nil
	}
	if ip.kind == NF3FREE {
		ip.put(txn)
		ip.releaseInode(txn)
		return nil
	}
	if ip.nlink == 0 {
		panic("getInodeInum")
	}
	return ip
}

func getInode(txn *txn, fh3 Nfs_fh3) *inode {
	fh := fh3.makeFh()
	ip := getInodeInum(txn, fh.ino)
	if ip == nil {
		return nil
	}
	if ip.gen != fh.gen {
		ip.put(txn)
		ip.releaseInode(txn)
		return nil
	}
	return ip
}

func (ip *inode) releaseInode(txn *txn) {
	addr := txn.fs.inum2addr(ip.inum)
	txn.release(addr)
}

func (ip *inode) writeInode(txn *txn) {
	if ip.inum >= txn.fs.nInode() {
		panic("writeInode")
	}
	buf := txn.readBufLocked(txn.fs.inum2addr(ip.inum), INODE)
	dPrintf(5, "writeInode %v\n", ip)
	ip.encode(buf)
	buf.setDirty()
}

func (txn *txn) allocInum() inum {
	n := txn.ialloc.allocNum(txn)
	dPrintf(5, "alloc inode %v\n", n)
	return inum(n)
}

func allocInode(txn *txn, kind Ftype3) inum {
	inum := txn.allocInum()
	if inum != 0 {
		ip := getInodeLocked(txn, inum)
		if ip.kind == NF3FREE {
			dPrintf(5, "allocInode: allocate inode %d\n", inum)
			ip.inum = inum
			ip.kind = kind
			ip.nlink = 1
			ip.gen = ip.gen + 1
		} else {
			panic("allocInode")
		}
		ip.writeInode(txn)

	}
	return inum
}

func (ip *inode) freeInode(txn *txn) {
	ip.kind = NF3FREE
	ip.gen = ip.gen + 1
	ip.writeInode(txn)
	txn.ialloc.freeNum(txn, uint64(ip.inum))
}

func freeInum(txn *txn, inum inum) {
	i := getInodeLocked(txn, inum)
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(txn)
}

// Done with ip and remove inode if nlink = 0.
func (ip *inode) put(txn *txn) {
	dPrintf(5, "put inode %d nlink %d\n", ip.inum, ip.nlink)
	// shrinker may put an FREE inode
	if ip.nlink == 0 && ip.kind != NF3FREE {
		ip.resize(txn, 0)
		ip.freeInode(txn)
	}
}

// Returns blkno for indirect bn and newroot if root was allocated. If
// blkno is 0, failure.
func (ip *inode) indbmap(txn *txn, root uint64, level uint64, off uint64, grow bool) (uint64, uint64) {
	var newroot uint64 = 0
	var blkno uint64 = root

	if blkno == 0 { // no root?
		newroot = txn.allocBlock()
		if newroot == 0 {
			return 0, 0
		}
		blkno = newroot
	}

	if level == 0 { // leaf?
		if root != 0 && grow { // old leaf?
			txn.zeroBlock(blkno)
		}
		return blkno, newroot
	}

	divisor := pow(level - 1)
	o := off / divisor
	bo := o * 8
	ind := off % divisor

	if root != 0 && off == 0 && grow { // old root from previous file?
		txn.zeroBlock(blkno)
	}

	buf := txn.readBlock(blkno)
	nxtroot := machine.UInt64Get(buf.blk[bo : bo+8])
	b, newroot1 := ip.indbmap(txn, nxtroot, level-1, ind, grow)
	if newroot1 != 0 {
		machine.UInt64Put(buf.blk[bo:bo+8], newroot1)
		buf.setDirty()
	}
	if b >= txn.fs.size {
		panic("indbmap")
	}
	return b, newroot
}

// Lazily resize file. Bmap allocates/zeros blocks on demand.  Create
// a new thread to free blocks in a separate transaction.
func (ip *inode) resize(txn *txn, sz uint64) {
	dPrintf(5, "resize %v to sz %d\n", ip, sz)
	if sz < ip.size {
		dPrintf(1, "start shrink thread\n")
		txn.nfs.nthread = txn.nfs.nthread + 1
		machine.Spawn(func() { shrink(txn.nfs, ip.inum, ip.size) })
	}
	ip.size = sz
	ip.writeInode(txn)
}

// Map logical block number bn to a physical block number, allocating
// blocks if no block exists for bn. Reuse block from previous
// versions of this inode, but zero them.
func (ip *inode) bmap(txn *txn, bn uint64) (uint64, bool) {
	var alloc bool = false
	sz := roundUp(ip.size, disk.BlockSize)
	grow := bn > sz
	if bn < NDIRECT {
		if ip.blks[bn] != 0 && grow {
			txn.zeroBlock(ip.blks[bn])
		}
		if ip.blks[bn] == 0 {
			blkno := txn.allocBlock()
			if blkno == 0 {
				return 0, false
			}
			alloc = true
			ip.blks[bn] = blkno
		}
		return ip.blks[bn], alloc
	} else {
		var off = bn - NDIRECT
		if off < NBLKBLK {
			blkno, newroot := ip.indbmap(txn, ip.blks[INDIRECT], 1, off, grow)
			if newroot != 0 {
				ip.blks[INDIRECT] = newroot
			}
			return blkno, newroot != 0
		} else {
			off -= NBLKBLK
			blkno, newroot := ip.indbmap(txn, ip.blks[DINDIRECT], 2, off, grow)
			if newroot != 0 {
				ip.blks[DINDIRECT] = newroot
			}
			return blkno, newroot != 0
		}
	}
}

// Returns number of bytes read and eof
func (ip *inode) read(txn *txn, offset uint64, bytesToRead uint64) ([]byte,
	bool) {
	var n uint64 = uint64(0)

	if offset >= ip.size {
		return nil, true
	}
	var count uint64 = bytesToRead
	if count >= offset+ip.size {
		count = ip.size - offset
	}
	var data = make([]byte, 0)
	var off = offset
	for boff := off / disk.BlockSize; n < count; boff++ {
		byteoff := off % disk.BlockSize
		nbytes := min(disk.BlockSize-byteoff, count-n)
		blkno, alloc := ip.bmap(txn, boff)
		if blkno == 0 {
			return data, false
		}
		if alloc { // fill in a hole
			ip.writeInode(txn)
		}
		buf := txn.readBlock(blkno)

		for b := uint64(0); b < nbytes; b++ {
			data = append(data, buf.blk[byteoff+b])
		}
		n += nbytes
		off += nbytes
	}
	return data, false
}

// Returns number of bytes written and error
func (ip *inode) write(txn *txn, offset uint64,
	count uint64,
	dataBuf []byte) (uint64, bool) {
	var cnt uint64 = uint64(0)
	var off uint64 = offset
	var ok bool = true
	var alloc bool = false
	var n = count
	var data = dataBuf

	if offset+count > maxFileSize() {
		return 0, false
	}
	for boff := off / disk.BlockSize; n > uint64(0); boff++ {
		blkno, new := ip.bmap(txn, boff)
		if blkno == 0 {
			ok = false
			break
		}
		if new {
			alloc = true
		}
		buf := txn.readBlock(blkno)
		byteoff := off % disk.BlockSize
		var nbytes = disk.BlockSize - byteoff
		if n < nbytes {
			nbytes = n
		}
		for b := uint64(0); b < nbytes; b++ {
			buf.blk[byteoff+b] = data[b]
		}
		buf.setDirty()
		n -= nbytes
		data = data[nbytes:]
		off += nbytes
		cnt += nbytes
	}
	if alloc || cnt > 0 {
		if off+cnt > ip.size {
			ip.size = off + cnt
		}
		ip.writeInode(txn)
	}
	return cnt, ok
}

func (ip *inode) decLink(txn *txn) {
	ip.nlink = ip.nlink - 1
	ip.writeInode(txn)
}

//
// Freeing of a file, run in separate thread/transaction
//

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *inode) indshrink(txn *txn, root uint64, level uint64, bn uint64) uint64 {
	if level == 0 {
		return root
	}
	divisor := pow(level - 1)
	off := (bn / divisor)
	ind := bn % divisor
	boff := off * 8
	buf := txn.readBlock(root)
	nxtroot := machine.UInt64Get(buf.blk[boff : boff+8])
	if nxtroot != 0 {
		freeroot := ip.indshrink(txn, nxtroot, level-1, ind)
		if freeroot != 0 {
			machine.UInt64Put(buf.blk[boff:boff+8], 0)
			buf.setDirty()
			txn.freeBlock(freeroot)
		}
	}
	if off == 0 && ind == 0 {
		return root
	} else {
		return 0
	}
}

func singletonTxn(ip *inode) []*inode {
	return []*inode{ip}
}

func shrink(nfs *Nfs, inum inum, oldsz uint64) {
	var bn = roundUp(oldsz, disk.BlockSize)
	dPrintf(1, "Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		txn := begin(nfs)
		ip := getInodeInumFree(txn, inum)
		if ip == nil {
			panic("shrink")
		}
		if ip.size >= oldsz { // file has grown again or resize didn't commit
			ok := txn.commit(singletonTxn(ip))
			if !ok {
				panic("shrink")
			}
			break
		}
		cursz := roundUp(ip.size, disk.BlockSize)
		dPrintf(5, "shrink: bn %d cursz %d\n", bn, cursz)
		// 4: inode block, 2xbitmap block, indirect block, double indirect
		for bn > cursz && txn.numberDirty()+4 < txn.log.logSz {
			bn = bn - 1
			if bn < NDIRECT {
				txn.freeBlock(ip.blks[bn])
				ip.blks[bn] = 0
			} else {
				var off = bn - NDIRECT
				if off < NBLKBLK {
					freeroot := ip.indshrink(txn, ip.blks[INDIRECT], 1, off)
					if freeroot != 0 {
						txn.freeBlock(ip.blks[INDIRECT])
						ip.blks[INDIRECT] = 0
					}
				} else {
					off = off - NBLKBLK
					freeroot := ip.indshrink(txn, ip.blks[DINDIRECT], 2, off)
					if freeroot != 0 {
						txn.freeBlock(ip.blks[DINDIRECT])
						ip.blks[DINDIRECT] = 0
					}
				}
			}
		}
		ip.writeInode(txn)
		ok := txn.commit(singletonTxn(ip))
		if !ok {
			panic("shrink")
		}
		if bn <= cursz {
			break
		}
	}
	dPrintf(1, "Shrinker: done shrinking %d to bn %d\n", inum, bn)
	nfs.mu.Lock()
	nfs.nthread = nfs.nthread - 1
	nfs.mu.Unlock()
	nfs.condShut.Signal()
}
