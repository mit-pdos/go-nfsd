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

type Inode struct {
	// in-memory info:
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

func (ip *Inode) encode(buf *Buf) {
	enc := NewEnc(buf.blk)
	enc.PutInt32(uint32(ip.kind))
	enc.PutInt32(ip.nlink)
	enc.PutInt(ip.gen)
	enc.PutInt(ip.size)
	enc.PutInts(ip.blks)
}

func decode(buf *Buf, inum uint64) *Inode {
	ip := &Inode{}
	dec := NewDec(buf.blk)
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

func getInodeLocked(txn *Txn, inum Inum) *Inode {
	addr := txn.fs.Inum2Addr(inum)
	buf := txn.ReadBufLocked(addr, INODE)
	i := decode(buf, inum)
	DPrintf(5, "getInodeLocked %v\n", i)
	return i
}

func getInodeInumFree(txn *Txn, inum Inum) *Inode {
	ip := getInodeLocked(txn, inum)
	return ip
}

func getInodeInum(txn *Txn, inum Inum) *Inode {
	ip := getInodeInumFree(txn, inum)
	if ip == nil {
		return nil
	}
	if ip.kind == NF3FREE {
		ip.put(txn)
		ip.ReleaseInode(txn)
		return nil
	}
	if ip.nlink == 0 {
		panic("getInodeInum")
	}
	return ip
}

func getInode(txn *Txn, fh3 Nfs_fh3) *Inode {
	fh := fh3.makeFh()
	ip := getInodeInum(txn, fh.ino)
	if ip == nil {
		return nil
	}
	if ip.gen != fh.gen {
		ip.put(txn)
		ip.ReleaseInode(txn)
		return nil
	}
	return ip
}

func (ip *Inode) ReleaseInode(txn *Txn) {
	addr := txn.fs.Inum2Addr(ip.inum)
	txn.ReleaseBuf(addr)
}

func (ip *Inode) writeInode(txn *Txn) {
	if ip.inum >= txn.fs.NInode() {
		panic("writeInode")
	}
	buf := txn.ReadBufLocked(txn.fs.Inum2Addr(ip.inum), INODE)
	DPrintf(5, "writeInode %v\n", ip)
	ip.encode(buf)
	buf.Dirty()
}

func (txn *Txn) AllocInum() Inum {
	n := txn.ialloc.AllocNum(txn)
	DPrintf(5, "alloc inode %v\n", n)
	return n
}

func allocInode(txn *Txn, kind Ftype3) Inum {
	inum := txn.AllocInum()
	if inum != 0 {
		ip := getInodeLocked(txn, inum)
		if ip.kind == NF3FREE {
			DPrintf(5, "allocInode: allocate inode %d\n", inum)
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

func (ip *Inode) freeInode(txn *Txn) {
	ip.kind = NF3FREE
	ip.gen = ip.gen + 1
	ip.writeInode(txn)
	txn.ialloc.FreeNum(txn, ip.inum)
}

func freeInum(txn *Txn, inum Inum) {
	i := getInodeLocked(txn, inum)
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(txn)
}

// Done with ip and remove inode if nlink = 0.
func (ip *Inode) put(txn *Txn) {
	DPrintf(5, "put inode %d nlink %d\n", ip.inum, ip.nlink)
	// shrinker may put an FREE inode
	if ip.nlink == 0 && ip.kind != NF3FREE {
		ip.resize(txn, 0)
		ip.freeInode(txn)
	}
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
			ZeroBlock(txn, blkno)
		}
		return blkno, newroot
	}

	divisor := pow(level - 1)
	o := off / divisor
	bo := o * 8
	ind := off % divisor

	if root != 0 && off == 0 && grow { // old root from previous file?
		ZeroBlock(txn, blkno)
	}

	buf := ReadBlock(txn, blkno)
	nxtroot := machine.UInt64Get(buf.blk[bo : bo+8])
	b, newroot1 := ip.indbmap(txn, nxtroot, level-1, ind, grow)
	if newroot1 != 0 {
		machine.UInt64Put(buf.blk[bo:bo+8], newroot1)
		buf.Dirty()
	}
	if b >= txn.fs.Size {
		panic("indbmap")
	}
	return b, newroot
}

// Lazily resize file. Bmap allocates/zeros blocks on demand.  Create
// a new thread to free blocks in a separate transaction.
func (ip *Inode) resize(txn *Txn, sz uint64) {
	DPrintf(5, "resize %v to sz %d\n", ip, sz)
	if sz < ip.size {
		DPrintf(1, "start shrink thread\n")
		txn.nfs.nthread = txn.nfs.nthread + 1
		go shrink(txn.nfs, ip.inum, ip.size)
	}
	ip.size = sz
	ip.writeInode(txn)
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
			ZeroBlock(txn, ip.blks[bn])
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
		buf := ReadBlock(txn, blkno)

		for b := uint64(0); b < nbytes; b++ {
			data = append(data, buf.blk[byteoff+b])
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
		buf := ReadBlock(txn, blkno)
		byteoff := offset % disk.BlockSize
		nbytes := disk.BlockSize - byteoff
		if n < nbytes {
			nbytes = n
		}
		for b := uint64(0); b < nbytes; b++ {
			buf.blk[byteoff+b] = data[b]
		}
		buf.Dirty()
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
	buf := ReadBlock(txn, root)
	nxtroot := machine.UInt64Get(buf.blk[boff : boff+8])
	if nxtroot != 0 {
		freeroot := ip.indshrink(txn, nxtroot, level-1, ind)
		if freeroot != 0 {
			machine.UInt64Put(buf.blk[boff:boff+8], 0)
			buf.Dirty()
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
	DPrintf(1, "Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		txn := Begin(nfs)
		ip := getInodeInumFree(txn, inum)
		if ip == nil {
			panic("shrink")
		}
		if ip.size >= oldsz { // file has grown again or resize didn't commit
			ok := txn.Commit([]*Inode{ip})
			if !ok {
				panic("shrink")
			}
			break
		}
		cursz := roundupblk(ip.size)
		DPrintf(5, "shrink: bn %d cursz %d\n", bn, cursz)
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
	DPrintf(1, "Shrinker: done shrinking %d to bn %d\n", inum, bn)
	nfs.mu.Lock()
	nfs.nthread = nfs.nthread - 1
	nfs.mu.Unlock()
	nfs.condShut.Signal()
}
