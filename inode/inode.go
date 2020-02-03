package inode

import (
	"fmt"
	"time"

	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/dcache"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

const NF3FREE nfstypes.Ftype3 = 0

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
	Inum   common.Inum
	Dcache *dcache.Dcache
	cslot  *cache.Cslot

	// the on-disk inode:
	Kind  nfstypes.Ftype3
	Nlink uint32
	Gen   uint64
	Size  uint64

	// if ShrinkSize > Size, then the inode is in the process
	// of shrinking to Size. ShrinkSize is in block units
	ShrinkSize uint64

	Atime nfstypes.Nfstime3
	Mtime nfstypes.Nfstime3
	blks  []common.Bnum
}

func NfstimeNow() nfstypes.Nfstime3 {
	now := time.Now()
	t := nfstypes.Nfstime3{
		Seconds:  nfstypes.Uint32(now.Unix()),
		Nseconds: nfstypes.Uint32(now.Nanosecond()),
	}
	return t
}

func (ip *Inode) initInode(inum common.Inum, kind nfstypes.Ftype3) {
	util.DPrintf(1, "initInode: inode # %d\n", inum)
	ip.Inum = inum
	ip.Kind = kind
	ip.Nlink = 1
	ip.Gen = ip.Gen + 1
	ip.Atime = NfstimeNow()
	ip.Mtime = NfstimeNow()
}

func MkRootInode() *Inode {
	i := &Inode{}
	i.blks = make([]common.Bnum, NBLKINO)
	i.initInode(common.ROOTINUM, nfstypes.NF3DIR)
	return i
}

func (ip *Inode) String() string {
	return fmt.Sprintf("# %d k %d n %d g %d sz %d ssz %d %v", ip.Inum, ip.Kind, ip.Nlink, ip.Gen, ip.Size, ip.ShrinkSize, ip.blks)
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
		Fileid: nfstypes.Fileid3(ip.Inum),
		Atime:  ip.Atime,
		Mtime:  ip.Mtime,
		Ctime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
	}
}

func (ip *Inode) Encode() []byte {
	enc := marshal.NewEnc(common.INODESZ)
	enc.PutInt32(uint32(ip.Kind))
	enc.PutInt32(ip.Nlink)
	enc.PutInt(ip.Gen)
	enc.PutInt(ip.Size)
	enc.PutInt(ip.ShrinkSize)
	enc.PutInt32(uint32(ip.Atime.Seconds))
	enc.PutInt32(uint32(ip.Atime.Nseconds))
	enc.PutInt32(uint32(ip.Mtime.Seconds))
	enc.PutInt32(uint32(ip.Mtime.Nseconds))
	enc.PutInts(ip.blks)
	return enc.Finish()
}

func Decode(buf *buf.Buf, inum common.Inum) *Inode {
	ip := &Inode{
		Inum:       0,
		Kind:       0,
		Nlink:      0,
		Gen:        0,
		Size:       0,
		ShrinkSize: 0,
		blks:       nil,
	}
	dec := marshal.NewDec(buf.Blk)
	ip.Inum = inum
	ip.Kind = nfstypes.Ftype3(dec.GetInt32())
	ip.Nlink = dec.GetInt32()
	ip.Gen = dec.GetInt()
	ip.Size = dec.GetInt()
	ip.ShrinkSize = dec.GetInt()
	ip.Atime.Seconds = nfstypes.Uint32(dec.GetInt32())
	ip.Atime.Nseconds = nfstypes.Uint32(dec.GetInt32())
	ip.Mtime.Seconds = nfstypes.Uint32(dec.GetInt32())
	ip.Mtime.Nseconds = nfstypes.Uint32(dec.GetInt32())
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

func (ip *Inode) ReleaseInode(op *FsTxn) {
	util.DPrintf(1, "ReleaseInode %v\n", ip)
	if ip.cslot == nil {
		panic("ReleaseInode")
	}
	ip.cslot.Unlock()
	op.doneInode(ip)
	op.Fs.Icache.FreeSlot(uint64(ip.Inum))
}

func LockInode(op *FsTxn, inum common.Inum) *cache.Cslot {
	cslot := op.Fs.Icache.LookupSlot(uint64(inum))
	if cslot == nil {
		panic("GetInodeLocked")
	}
	cslot.Lock()
	return cslot
}

// XXX add LookupRef
func GetInodeUnlocked(op *FsTxn, inum common.Inum) *Inode {
	cslot := op.Fs.Icache.LookupSlot(uint64(inum))
	if cslot == nil || cslot.Obj == nil {
		panic("GetInodeUnlocked")
	}
	ip := cslot.Obj.(*Inode)
	op.Fs.Icache.FreeSlot(uint64(ip.Inum))
	return ip
}

func GetInodeLocked(op *FsTxn, inum common.Inum) *Inode {
	cslot := LockInode(op, inum)
	if cslot.Obj == nil {
		addr := op.Fs.Super.Inum2Addr(inum)
		buf := op.buftxn.ReadBuf(addr)
		i := Decode(buf, inum)
		util.DPrintf(1, "GetInodeLocked # %v: read inode from disk\n", inum)
		cslot.Obj = i
	}
	ip := cslot.Obj.(*Inode)
	ip.cslot = cslot
	op.addInode(ip)
	util.DPrintf(1, "%d: GetInodeLocked %v\n", op.buftxn.Id, ip)
	return ip
}

func getInodeInumFree(op *FsTxn, inum common.Inum) *Inode {
	ip := GetInodeLocked(op, inum)
	return ip
}

func GetInodeInum(op *FsTxn, inum common.Inum) *Inode {
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

func GetInodeFh(op *FsTxn, fh3 nfstypes.Nfs_fh3) *Inode {
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

func (ip *Inode) WriteInode(op *FsTxn) {
	if ip.Inum >= op.Fs.Super.NInode() {
		panic("WriteInode")
	}
	d := ip.Encode()
	op.buftxn.OverWrite(op.Fs.Super.Inum2Addr(ip.Inum), d)
	util.DPrintf(1, "WriteInode %v\n", ip)
}

func AllocInode(op *FsTxn, kind nfstypes.Ftype3) (*FsTxn, *Inode, bool) {
	var ip *Inode
	var ok bool = true
	inum := common.Inum(op.Fs.Ialloc.AllocNum())
	if inum != common.NULLINUM {
		ip = GetInodeLocked(op, inum)
		if ip.Kind != NF3FREE {
			panic("AllocInode")
		}
		if ip.IsShrinking() {
			// give the number back so that if HelpShrink()
			// commits, it doesn't mark inum as allocated.
			op.Fs.Ialloc.FreeNum(uint64(inum))
			op, ok = HelpShrink(op, ip)
			ip = nil
		} else {
			util.DPrintf(1, "AllocInode -> # %v\n", inum)
			op.AddINum(inum)
			ip.initInode(inum, kind)
			ip.WriteInode(op)
		}
	}
	return op, ip, ok
}

func (ip *Inode) freeInode(op *FsTxn) {
	ip.Kind = NF3FREE
	ip.Gen = ip.Gen + 1
	ip.WriteInode(op)
	op.FreeINum(ip.Inum)
}

func FreeInum(op *FsTxn, inum common.Inum) {
	i := GetInodeLocked(op, inum)
	if i.Kind == NF3FREE {
		panic("freeInode")
	}
	i.freeInode(op)
}

// Done with ip and remove inode if Nlink = 0.
func (ip *Inode) Put(op *FsTxn) {
	util.DPrintf(1, "put inode # %d Nlink %d\n", ip.Inum, ip.Nlink)
	// shrinker may put an FREE inode
	if ip.Nlink == 0 && ip.Kind != NF3FREE {
		ip.Resize(op, 0)
		ip.freeInode(op)
	}
}

// Resize updates the inode, but may not free immediately if the inode
// shrinks. It creates a new thread to free blocks in a separate
// transaction, if shrinking involves freeing many blocks.  ShrinkSize
// tracks shrinking progress, and is initialized with the old size.
func (ip *Inode) Resize(op *FsTxn, sz uint64) {
	oldsz := util.RoundUp(ip.Size, disk.BlockSize)
	util.DPrintf(5, "Resize %v to sz %d\n", oldsz, sz)
	ip.Size = sz
	sz = util.RoundUp(sz, disk.BlockSize)
	if sz < oldsz {
		ip.ShrinkSize = oldsz
	} else {
		ip.ShrinkSize = sz
	}
	ip.WriteInode(op)
	if sz < oldsz {
		if ip.shrinkFits(op) {
			ip.Shrink(op)
			util.DPrintf(1, "small file delete inside trans\n")
		} else {
			// for large files, start a separate thread
			util.DPrintf(1, "start shrink thread\n")
			shrinkst.mu.Lock()
			shrinkst.nthread = shrinkst.nthread + 1
			shrinkst.mu.Unlock()
			machine.Spawn(func() { shrinker(ip.Inum) })
		}
	}
}

// Returns blkno and root index block for off. If blkno is 0, failure.
// Caller must compare root with returned root to decide if a root has
// been allocated.
func (ip *Inode) indbmap(op *FsTxn, root common.Bnum, level uint64, off uint64) (common.Bnum, common.Bnum) {
	if root == common.NULLBNUM { // no root?
		root = op.AllocBlock()
		if root == common.NULLBNUM {
			return root, root
		}
	}
	if level == 0 { // leaf?
		return root, root
	}

	divisor := pow(level - 1)
	o := off / divisor
	bo := o * 8
	ind := off % divisor

	buf := op.ReadBlock(root)
	nxtroot := buf.BnumGet(bo)
	util.DPrintf(1, "%d next root %v level %d\n", root, nxtroot, level)
	blkno, newnextroot := ip.indbmap(op, nxtroot, level-1, ind)
	op.AssertValidBlock(newnextroot)
	op.AssertValidBlock(blkno)
	if newnextroot != nxtroot {
		buf.BnumPut(bo, newnextroot)
	}
	return blkno, root
}

// Map logical block number bn to a physical block number, allocating
// blocks if no block exists for bn.
func (ip *Inode) bmap(op *FsTxn, bn uint64) (common.Bnum, bool) {
	var blkno = common.NULLBNUM
	var alloc = false
	if bn < NDIRECT {
		if ip.blks[bn] == common.NULLBNUM {
			ip.blks[bn] = op.AllocBlock()
			if ip.blks[bn] != common.NULLBNUM {
				alloc = true
			}
		}
		blkno = ip.blks[bn]
	} else {
		var off = bn - NDIRECT
		var root = common.NULLBNUM
		if off < NBLKBLK {
			blkno, root = ip.indbmap(op, ip.blks[INDIRECT], 1, off)
			alloc = root != ip.blks[INDIRECT]
			if alloc {
				ip.blks[INDIRECT] = root
			}
		} else {
			off -= NBLKBLK
			blkno, root = ip.indbmap(op, ip.blks[DINDIRECT], 2, off)
			alloc = root != ip.blks[INDIRECT]
			if alloc {
				ip.blks[DINDIRECT] = root
			}
		}
	}
	return blkno, alloc
}

// Returns number of bytes read and eof
func (ip *Inode) Read(op *FsTxn, offset uint64, bytesToRead uint64) ([]byte,
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
		if blkno == common.NULLBNUM {
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
	util.DPrintf(10, "Read: off %d cnt %d -> %v\n", offset, count, data)
	return data, false
}

// Returns number of bytes written and error
func (ip *Inode) Write(op *FsTxn, offset uint64,
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
		if blkno == common.NULLBNUM {
			ok = false
			break
		}
		if new {
			alloc = true
		}
		byteoff := off % disk.BlockSize
		var nbytes = disk.BlockSize - byteoff
		if n < nbytes {
			nbytes = n
		}
		if byteoff == 0 && nbytes == disk.BlockSize { // block overwrite?
			addr := op.Fs.Super.Block2addr(blkno)
			op.buftxn.OverWrite(addr, data[0:nbytes])
		} else {
			buffer := op.ReadBlock(blkno)
			for b := uint64(0); b < nbytes; b++ {
				buffer.Blk[byteoff+b] = data[b]
			}
			buffer.SetDirty()
		}
		n -= nbytes
		data = data[nbytes:]
		off += nbytes
		cnt += nbytes
	}
	util.DPrintf(1, "Write: off %d cnt %d size %d\n", offset, cnt, ip.Size)
	if alloc || cnt > 0 {
		if offset+cnt > ip.Size {
			ip.Size = offset + cnt
		}
		ip.WriteInode(op)
		return cnt, true
	}
	return cnt, ok
}

func (ip *Inode) DecLink(op *FsTxn) {
	ip.Nlink = ip.Nlink - 1
	ip.WriteInode(op)
}
