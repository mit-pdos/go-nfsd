package inode

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/dcache"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/marshal"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"

	"fmt"
)

const NF3FREE nfstypes.Ftype3 = 0

const (
	NBLKINO   uint64 = 5 // # blk in an inode's blks array
	NDIRECT   uint64 = NBLKINO - 2
	INDIRECT  uint64 = NBLKINO - 2
	DINDIRECT uint64 = NBLKINO - 1
	NBLKBLK   uint64 = disk.BlockSize / 8 // # blkno per block
	NINDLEVEL uint64 = 2                  // # levels of indirection
)

type Inode struct {
	// in-memory info:
	Inum   fs.Inum
	Dcache *dcache.Dcache

	// the on-disk inode:
	Kind  nfstypes.Ftype3
	Nlink uint32
	Gen   uint64
	Size  uint64
	blks  []uint64
}

func MkNullInode() *Inode {
	return &Inode{
		Inum:  fs.NULLINUM,
		Kind:  nfstypes.NF3DIR,
		Nlink: uint32(1),
		Gen:   uint64(0),
		Size:  uint64(0),
		blks:  make([]uint64, NBLKINO),
	}
}

func MkRootInode() *Inode {
	return &Inode{
		Inum:  fs.ROOTINUM,
		Kind:  nfstypes.NF3DIR,
		Nlink: uint32(1),
		Gen:   uint64(0),
		Size:  uint64(0),
		blks:  make([]uint64, NBLKINO),
	}
}

func (ip *Inode) String() string {
	return fmt.Sprintf("# %d k %d n %d g %d sz %d %v", ip.Inum, ip.Kind, ip.Nlink, ip.Gen, ip.Size, ip.blks)
}

func (ip *Inode) MkFattr() nfstypes.Fattr3 {
	return nfstypes.Fattr3{
		Ftype: ip.Kind,
		Mode:  0777,
		Nlink: 1,
		Uid:   nfstypes.Uid3(0),
		Gid:   nfstypes.Gid3(0),
		Size:  nfstypes.Size3(ip.Size),
		Used:  nfstypes.Size3(ip.Size),
		Rdev: nfstypes.Specdata3{Specdata1: nfstypes.Uint32(0),
			Specdata2: nfstypes.Uint32(0)},
		Fsid:   nfstypes.Uint64(0),
		Fileid: nfstypes.Fileid3(0),
		Atime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Mtime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Ctime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
	}
}

func (ip *Inode) Encode(buf *buf.Buf) {
	enc := marshal.NewEnc(buf.Blk)
	enc.PutInt32(uint32(ip.Kind))
	enc.PutInt32(ip.Nlink)
	enc.PutInt(ip.Gen)
	enc.PutInt(ip.Size)
	enc.PutInts(ip.blks)
}

func decode(buf *buf.Buf, inum fs.Inum) *Inode {
	ip := &Inode{
		Inum:  0,
		Kind:  0,
		Nlink: 0,
		Gen:   0,
		Size:  0,
		blks:  nil,
	}
	dec := marshal.NewDec(buf.Blk)
	ip.Inum = inum
	ip.Kind = nfstypes.Ftype3(dec.GetInt32())
	ip.Nlink = dec.GetInt32()
	ip.Gen = dec.GetInt()
	ip.Size = dec.GetInt()
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

func GetInodeLocked(op *fstxn.FsTxn, inum fs.Inum) *Inode {
	addr := op.Fs.Inum2Addr(inum)
	op.Acquire(addr)
	cslot := op.LookupSlot(inum)
	if cslot == nil {
		panic("GetInodeLocked")
	}
	if cslot.Obj == nil {
		buf := op.ReadBuf(addr)
		i := decode(buf, inum)
		cslot.Obj = i
	}
	i := cslot.Obj.(*Inode)
	util.DPrintf(5, "getInodeLocked %v\n", i)
	return i
}

func getInodeInumFree(op *fstxn.FsTxn, inum fs.Inum) *Inode {
	ip := GetInodeLocked(op, inum)
	return ip
}

func GetInodeInum(op *fstxn.FsTxn, inum fs.Inum) *Inode {
	ip := getInodeInumFree(op, inum)
	if ip == nil {
		return nil
	}
	if ip.Kind == NF3FREE {
		ip.Put(op)
		ip.ReleaseInode(op)
		return nil
	}
	if ip.Nlink == 0 {
		panic("getInodeInum")
	}
	return ip
}

func GetInode(op *fstxn.FsTxn, fh3 nfstypes.Nfs_fh3) *Inode {
	fh := fh.MakeFh(fh3)
	ip := GetInodeInum(op, fh.Ino)
	if ip == nil {
		return nil
	}
	if ip.Gen != fh.Gen {
		ip.Put(op)
		ip.ReleaseInode(op)
		return nil
	}
	return ip
}

func (ip *Inode) ReleaseInode(op *fstxn.FsTxn) {
	addr := op.Fs.Inum2Addr(ip.Inum)
	op.Release(addr)
}

func (ip *Inode) WriteInode(op *fstxn.FsTxn) {
	if ip.Inum >= op.Fs.NInode() {
		panic("WriteInode")
	}
	buf := op.LookupBuf(op.Fs.Inum2Addr(ip.Inum))
	ip.Encode(buf)
	buf.SetDirty()
	util.DPrintf(1, "WriteInode %v %v\n", ip, buf)
}

func AllocInode(op *fstxn.FsTxn, kind nfstypes.Ftype3) (fs.Inum, *Inode) {
	var ip *Inode
	inum := op.AllocINum()
	if inum != 0 {
		ip = GetInodeLocked(op, inum)
		if ip.Kind == NF3FREE {
			util.DPrintf(5, "allocInode: allocate inode %d\n", inum)
			ip.Inum = inum
			ip.Kind = kind
			ip.Nlink = 1
			ip.Gen = ip.Gen + 1
		} else {
			panic("allocInode")
		}
		ip.WriteInode(op)
	}
	return inum, ip
}

func (ip *Inode) freeInode(op *fstxn.FsTxn) {
	ip.Kind = NF3FREE
	ip.Gen = ip.Gen + 1
	ip.WriteInode(op)
	op.FreeINum(ip.Inum)
}

func FreeInum(op *fstxn.FsTxn, inum fs.Inum) {
	i := GetInodeLocked(op, inum)
	if i.Kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(op)
}

// Done with ip and remove inode if Nlink = 0.
func (ip *Inode) Put(op *fstxn.FsTxn) {
	util.DPrintf(5, "put inode %d Nlink %d\n", ip.Inum, ip.Nlink)
	// shrinker may put an FREE inode
	if ip.Nlink == 0 && ip.Kind != NF3FREE {
		ip.Resize(op, 0)
		ip.freeInode(op)
	}
	op.FreeSlot(ip.Inum)
}

func putInodes(op *fstxn.FsTxn, inodes []*Inode) {
	for _, ip := range inodes {
		ip.Put(op)
	}
}

// Returns blkno for indirect bn and newroot if root was allocated. If
// blkno is 0, failure.
func (ip *Inode) indbmap(op *fstxn.FsTxn, root uint64, level uint64, off uint64, grow bool) (uint64, uint64) {
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
func (ip *Inode) Resize(op *fstxn.FsTxn, sz uint64) {
	util.DPrintf(5, "resize %v to sz %d\n", ip, sz)
	if sz < ip.Size {
		util.DPrintf(1, "start shrink thread\n")
		shrinker.nthread = shrinker.nthread + 1
		machine.Spawn(func() { shrink(ip.Inum, ip.Size) })
	}
	ip.Size = sz
	ip.WriteInode(op)
}

// Map logical block number bn to a physical block number, allocating
// blocks if no block exists for bn. Reuse block from previous
// versions of this inode, but zero them.
func (ip *Inode) bmap(op *fstxn.FsTxn, bn uint64) (uint64, bool) {
	var alloc bool = false
	sz := util.RoundUp(ip.Size, disk.BlockSize)
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
func (ip *Inode) Read(op *fstxn.FsTxn, offset uint64, bytesToRead uint64) ([]byte,
	bool) {
	var n uint64 = uint64(0)

	if offset >= ip.Size {
		return nil, true
	}
	var count uint64 = bytesToRead
	if count >= offset+ip.Size {
		count = ip.Size - offset
	}
	util.DPrintf(5, "Read: off %d cnt %d\n", offset, count)
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
			ip.WriteInode(op)
		}
		buf := op.ReadBlock(blkno)

		for b := uint64(0); b < nbytes; b++ {
			data = append(data, buf.Blk[byteoff+b])
		}
		n += nbytes
		off += nbytes
	}
	util.DPrintf(5, "Read: off %d cnt %d -> %v\n", offset, count, data)
	return data, false
}

// Returns number of bytes written and error
func (ip *Inode) Write(op *fstxn.FsTxn, offset uint64,
	count uint64, dataBuf []byte) (uint64, bool) {
	var cnt uint64 = uint64(0)
	var off uint64 = offset
	var ok bool = true
	var alloc bool = false
	var n = count
	var data = dataBuf

	util.DPrintf(5, "Write: off %d cnt %d\n", offset, count)
	if offset+count > MaxFileSize() {
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
	util.DPrintf(5, "Write: off %d cnt %d size %d\n", offset, cnt, ip.Size)
	if alloc || cnt > 0 {
		if offset+cnt > ip.Size {
			ip.Size = offset + cnt
		}
		ip.WriteInode(op)
	}
	return cnt, ok
}

func (ip *Inode) DecLink(op *fstxn.FsTxn) {
	ip.Nlink = ip.Nlink - 1
	ip.WriteInode(op)
}

//
// Freeing of a file, run in separate thread/transaction
//

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *Inode) indshrink(op *fstxn.FsTxn, root uint64, level uint64, bn uint64) uint64 {
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
