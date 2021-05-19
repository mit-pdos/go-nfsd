package simple

import (
	"github.com/tchajed/goose/machine/disk"
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/go-journal/buf"
	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-journal/jrnl"
	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

type Inode struct {
	Inum common.Inum
	Size uint64 // 0 to 4KB
	Data common.Bnum
}

func (ip *Inode) Encode() []byte {
	enc := marshal.NewEnc(common.INODESZ)
	enc.PutInt(ip.Size)
	enc.PutInt(ip.Data)
	return enc.Finish()
}

func Decode(buf *buf.Buf, inum common.Inum) *Inode {
	ip := new(Inode)
	dec := marshal.NewDec(buf.Data)
	ip.Inum = inum
	ip.Size = dec.GetInt()
	ip.Data = dec.GetInt()
	return ip
}

// Returns number of bytes read and eof
func (ip *Inode) Read(op *jrnl.Op, offset uint64,
	bytesToRead uint64) ([]byte, bool) {
	if offset >= ip.Size {
		return nil, true
	}
	var count uint64 = bytesToRead
	if count > ip.Size-offset {
		count = ip.Size - offset
	}
	util.DPrintf(5, "Read: off %d cnt %d\n", offset, count)
	var data = make([]byte, 0)

	buf := op.ReadBuf(block2addr(ip.Data), common.NBITBLOCK)
	countCopy := count
	for b := uint64(0); b < countCopy; b++ {
		data = append(data, buf.Data[offset+b])
	}

	eof := (offset+count >= ip.Size)

	util.DPrintf(10, "Read: off %d cnt %d -> %v, %v\n", offset, count, data, eof)
	return data, eof
}

func (ip *Inode) WriteInode(op *jrnl.Op) {
	d := ip.Encode()
	op.OverWrite(inum2Addr(ip.Inum), common.INODESZ*8, d)
	util.DPrintf(1, "WriteInode %v\n", ip)
}

// Returns number of bytes written and error
func (ip *Inode) Write(op *jrnl.Op, offset uint64, count uint64,
	dataBuf []byte) (uint64, bool) {
	util.DPrintf(5, "Write: off %d cnt %d\n", offset, count)
	if count != uint64(len(dataBuf)) {
		return 0, false
	}

	if util.SumOverflows(offset, count) {
		return 0, false
	}

	if offset+count > disk.BlockSize {
		return 0, false
	}

	if offset > ip.Size {
		return 0, false
	}

	buffer := op.ReadBuf(block2addr(ip.Data), common.NBITBLOCK)
	for b := uint64(0); b < count; b++ {
		buffer.Data[offset+b] = dataBuf[b]
	}
	buffer.SetDirty()

	util.DPrintf(1, "Write: off %d cnt %d size %d\n", offset, count, ip.Size)
	if offset+count > ip.Size {
		ip.Size = offset + count
		ip.WriteInode(op)
	}
	return count, true
}

func ReadInode(op *jrnl.Op, inum common.Inum) *Inode {
	buffer := op.ReadBuf(inum2Addr(inum), common.INODESZ*8)
	ip := Decode(buffer, inum)
	return ip
}

func (ip *Inode) MkFattr() nfstypes.Fattr3 {
	return nfstypes.Fattr3{
		Ftype: nfstypes.NF3REG,
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
		Atime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Mtime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Ctime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
	}
}

func inodeInit(op *jrnl.Op) {
	for i := common.Inum(0); i < nInode(); i++ {
		ip := ReadInode(op, i)
		ip.Data = common.LOGSIZE + 1 + i
		ip.WriteInode(op)
	}
}
