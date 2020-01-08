package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/marshal"
	"github.com/mit-pdos/goose-nfsd/util"

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
)

const NULLINUM fs.Inum = 0
const ROOTINUM fs.Inum = 1

type inode struct {
	// in-memory info:
	inum fs.Inum

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

func (ip *inode) encode(buf *buf.Buf) {
	enc := marshal.NewEnc(buf.Blk)
	enc.PutInt32(uint32(ip.kind))
	enc.PutInt32(ip.nlink)
	enc.PutInt(ip.gen)
	enc.PutInt(ip.size)
	enc.PutInts(ip.blks)
}

func decode(buf *buf.Buf, inum fs.Inum) *inode {
	ip := &inode{
		inum:  0,
		kind:  0,
		nlink: 0,
		gen:   0,
		size:  0,
		blks:  nil,
	}
	dec := marshal.NewDec(buf.Blk)
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

func maxFileSize() uint64 {
	maxblks := pow(NINDLEVEL)
	return (NDIRECT + maxblks) * disk.BlockSize
}

func getInodeLocked(op *fstxn.FsTxn, inum fs.Inum) *inode {
	addr := op.Fs.Inum2Addr(inum)
	buf := op.ReadBufLocked(addr)
	i := decode(buf, inum)
	util.DPrintf(5, "getInodeLocked %v\n", i)
	return i
}

func getInodeInumFree(op *fstxn.FsTxn, inum fs.Inum) *inode {
	ip := getInodeLocked(op, inum)
	return ip
}

func getInodeInum(op *fstxn.FsTxn, inum fs.Inum) *inode {
	ip := getInodeInumFree(op, inum)
	if ip == nil {
		return nil
	}
	if ip.kind == NF3FREE {
		ip.put(op)
		ip.releaseInode(op)
		return nil
	}
	if ip.nlink == 0 {
		panic("getInodeInum")
	}
	return ip
}

func getInode(op *fstxn.FsTxn, fh3 Nfs_fh3) *inode {
	fh := fh3.makeFh()
	ip := getInodeInum(op, fh.ino)
	if ip == nil {
		return nil
	}
	if ip.gen != fh.gen {
		ip.put(op)
		ip.releaseInode(op)
		return nil
	}
	return ip
}

func (ip *inode) releaseInode(op *fstxn.FsTxn) {
	addr := op.Fs.Inum2Addr(ip.inum)
	op.Release(addr)
}

func (ip *inode) writeInode(op *fstxn.FsTxn) {
	if ip.inum >= op.Fs.NInode() {
		panic("writeInode")
	}
	buf := op.ReadBufLocked(op.Fs.Inum2Addr(ip.inum))
	util.DPrintf(5, "writeInode %v\n", ip)
	ip.encode(buf)
	buf.SetDirty()
}

func allocInum(op *fstxn.FsTxn) fs.Inum {
	n := op.AllocINum()
	util.DPrintf(5, "alloc inode %v\n", n)
	return fs.Inum(n)
}

func allocInode(op *fstxn.FsTxn, kind Ftype3) fs.Inum {
	inum := op.AllocINum()
	if inum != 0 {
		ip := getInodeLocked(op, inum)
		if ip.kind == NF3FREE {
			util.DPrintf(5, "allocInode: allocate inode %d\n", inum)
			ip.inum = inum
			ip.kind = kind
			ip.nlink = 1
			ip.gen = ip.gen + 1
		} else {
			panic("allocInode")
		}
		ip.writeInode(op)

	}
	return inum
}

func (ip *inode) freeInode(op *fstxn.FsTxn) {
	ip.kind = NF3FREE
	ip.gen = ip.gen + 1
	ip.writeInode(op)
	op.FreeINum(ip.inum)
}

func freeInum(op *fstxn.FsTxn, inum fs.Inum) {
	i := getInodeLocked(op, inum)
	if i.kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(op)
}

// Done with ip and remove inode if nlink = 0.
func (ip *inode) put(op *fstxn.FsTxn) {
	util.DPrintf(5, "put inode %d nlink %d\n", ip.inum, ip.nlink)
	// shrinker may put an FREE inode
	if ip.nlink == 0 && ip.kind != NF3FREE {
		ip.resize(op, 0)
		ip.freeInode(op)
	}
}

func putInodes(op *fstxn.FsTxn, inodes []*inode) {
	for _, ip := range inodes {
		ip.put(op)
	}
}

// Returns blkno for indirect bn and newroot if root was allocated. If
// blkno is 0, failure.
func (ip *inode) indbmap(op *fstxn.FsTxn, root uint64, level uint64, off uint64, grow bool) (uint64, uint64) {
	var newroot uint64 = 0
	var blkno uint64 = root

	if blkno == 0 { // no root?
		newroot = op.AllocBlock()
		if newroot == 0 {
			return 0, 0
		}
		blkno = newroot
	}

	if level == 0 { // leaf?
		if root != 0 && grow { // old leaf?
			op.ZeroBlock(blkno)
		}
		return blkno, newroot
	}

	divisor := pow(level - 1)
	o := off / divisor
	bo := o * 8
	ind := off % divisor

	if root != 0 && off == 0 && grow { // old root from previous file?
		op.ZeroBlock(blkno)
	}

	buf := op.ReadBlock(blkno)
	nxtroot := machine.UInt64Get(buf.Blk[bo : bo+8])
	b, newroot1 := ip.indbmap(op, nxtroot, level-1, ind, grow)
	if newroot1 != 0 {
		machine.UInt64Put(buf.Blk[bo:bo+8], newroot1)
		buf.SetDirty()
	}
	if b >= op.Fs.Size {
		panic("indbmap")
	}
	return b, newroot
}

// Lazily resize file. Bmap allocates/zeros blocks on demand.  Create
// a new thread to free blocks in a separate transaction.
func (ip *inode) resize(op *fstxn.FsTxn, sz uint64) {
	util.DPrintf(5, "resize %v to sz %d\n", ip, sz)
	if sz < ip.size {
		util.DPrintf(1, "start shrink thread\n")
		nfs.nthread = nfs.nthread + 1
		machine.Spawn(func() { shrink(ip.inum, ip.size) })
	}
	ip.size = sz
	ip.writeInode(op)
}

// Map logical block number bn to a physical block number, allocating
// blocks if no block exists for bn. Reuse block from previous
// versions of this inode, but zero them.
func (ip *inode) bmap(op *fstxn.FsTxn, bn uint64) (uint64, bool) {
	var alloc bool = false
	sz := util.RoundUp(ip.size, disk.BlockSize)
	grow := bn > sz
	if bn < NDIRECT {
		if ip.blks[bn] != 0 && grow {
			op.ZeroBlock(ip.blks[bn])
		}
		if ip.blks[bn] == 0 {
			blkno := op.AllocBlock()
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
			blkno, newroot := ip.indbmap(op, ip.blks[INDIRECT], 1, off, grow)
			if newroot != 0 {
				ip.blks[INDIRECT] = newroot
			}
			return blkno, newroot != 0
		} else {
			off -= NBLKBLK
			blkno, newroot := ip.indbmap(op, ip.blks[DINDIRECT], 2, off, grow)
			if newroot != 0 {
				ip.blks[DINDIRECT] = newroot
			}
			return blkno, newroot != 0
		}
	}
}

// Returns number of bytes read and eof
func (ip *inode) read(op *fstxn.FsTxn, offset uint64, bytesToRead uint64) ([]byte,
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
		nbytes := util.Min(disk.BlockSize-byteoff, count-n)
		blkno, alloc := ip.bmap(op, boff)
		if blkno == 0 {
			return data, false
		}
		if alloc { // fill in a hole
			ip.writeInode(op)
		}
		buf := op.ReadBlock(blkno)

		for b := uint64(0); b < nbytes; b++ {
			data = append(data, buf.Blk[byteoff+b])
		}
		n += nbytes
		off += nbytes
	}
	return data, false
}

// Returns number of bytes written and error
func (ip *inode) write(op *fstxn.FsTxn, offset uint64,
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
		blkno, new := ip.bmap(op, boff)
		if blkno == 0 {
			ok = false
			break
		}
		if new {
			alloc = true
		}
		buf := op.ReadBlock(blkno)
		byteoff := off % disk.BlockSize
		var nbytes = disk.BlockSize - byteoff
		if n < nbytes {
			nbytes = n
		}
		for b := uint64(0); b < nbytes; b++ {
			buf.Blk[byteoff+b] = data[b]
		}
		buf.SetDirty()
		n -= nbytes
		data = data[nbytes:]
		off += nbytes
		cnt += nbytes
	}
	if alloc || cnt > 0 {
		if off+cnt > ip.size {
			ip.size = off + cnt
		}
		ip.writeInode(op)
	}
	return cnt, ok
}

func (ip *inode) decLink(op *fstxn.FsTxn) {
	ip.nlink = ip.nlink - 1
	ip.writeInode(op)
}

//
// Freeing of a file, run in separate thread/transaction
//

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *inode) indshrink(op *fstxn.FsTxn, root uint64, level uint64, bn uint64) uint64 {
	if level == 0 {
		return root
	}
	divisor := pow(level - 1)
	off := (bn / divisor)
	ind := bn % divisor
	boff := off * 8
	buf := op.ReadBlock(root)
	nxtroot := machine.UInt64Get(buf.Blk[boff : boff+8])
	if nxtroot != 0 {
		freeroot := ip.indshrink(op, nxtroot, level-1, ind)
		if freeroot != 0 {
			machine.UInt64Put(buf.Blk[boff:boff+8], 0)
			buf.SetDirty()
			op.FreeBlock(freeroot)
		}
	}
	if off == 0 && ind == 0 {
		return root
	} else {
		return 0
	}
}

func singletonTrans(ip *inode) []*inode {
	return []*inode{ip}
}

func shrink(inum fs.Inum, oldsz uint64) {
	var bn = util.RoundUp(oldsz, disk.BlockSize)
	util.DPrintf(1, "Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		op := fstxn.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
		ip := getInodeInumFree(op, inum)
		if ip == nil {
			panic("shrink")
		}
		if ip.size >= oldsz { // file has grown again or resize didn't commit
			ok := commit(op, singletonTrans(ip))
			if !ok {
				panic("shrink")
			}
			break
		}
		cursz := util.RoundUp(ip.size, disk.BlockSize)
		util.DPrintf(5, "shrink: bn %d cursz %d\n", bn, cursz)
		// 4: inode block, 2xbitmap block, indirect block, double indirect
		for bn > cursz && op.NumberDirty()+4 < op.LogSz() {
			bn = bn - 1
			if bn < NDIRECT {
				op.FreeBlock(ip.blks[bn])
				ip.blks[bn] = 0
			} else {
				var off = bn - NDIRECT
				if off < NBLKBLK {
					freeroot := ip.indshrink(op, ip.blks[INDIRECT], 1, off)
					if freeroot != 0 {
						op.FreeBlock(ip.blks[INDIRECT])
						ip.blks[INDIRECT] = 0
					}
				} else {
					off = off - NBLKBLK
					freeroot := ip.indshrink(op, ip.blks[DINDIRECT], 2, off)
					if freeroot != 0 {
						op.FreeBlock(ip.blks[DINDIRECT])
						ip.blks[DINDIRECT] = 0
					}
				}
			}
		}
		ip.writeInode(op)
		ok := commit(op, singletonTrans(ip))
		if !ok {
			panic("shrink")
		}
		if bn <= cursz {
			break
		}
	}
	util.DPrintf(1, "Shrinker: done shrinking %d to bn %d\n", inum, bn)
	nfs.mu.Lock()
	nfs.nthread = nfs.nthread - 1
	nfs.mu.Unlock()
	nfs.condShut.Signal()
}
