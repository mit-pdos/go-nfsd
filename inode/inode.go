package inode

import (
	"time"

	// "github.com/tchajed/goose/machine/disk"
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

const NF3FREE nfstypes.Ftype3 = 0

const (
	NENTRIES  uint64 = 8
)

type Inode struct {
	// in-memory info:
	Inum   common.Inum

	// the on-disk inode:
	Gen      uint64
	Parent   common.Inum
	Contents []common.Inum
	Names    []byte
}

func NfstimeNow() nfstypes.Nfstime3 {
	now := time.Now()
	t := nfstypes.Nfstime3{
		Seconds:  nfstypes.Uint32(now.Unix()),
		Nseconds: nfstypes.Uint32(now.Nanosecond()),
	}
	return t
}

func (ip *Inode) InitInode(inum common.Inum, parent common.Inum) {
	util.DPrintf(1, "initInode: inode # %d\n", inum)
	ip.Inum = inum
	ip.Parent = parent
	ip.Gen = ip.Gen + 1
	ip.Contents = make([]common.Inum, NENTRIES)
	ip.Names = make([]byte, NENTRIES)
}

func MkRootInode() *Inode {
	ip := new(Inode)
	ip.InitInode(common.ROOTINUM, 0)
	return ip
}

func (ip *Inode) MkFh3() nfstypes.Nfs_fh3 {
	return fh.Fh{
		Ino: ip.Inum,
		Gen: ip.Gen,
	}.MakeFh3()
}

func (ip *Inode) MkFattr() nfstypes.Fattr3 {
	return nfstypes.Fattr3{
		Ftype: nfstypes.NF3DIR,
		Mode:  0777,
		Nlink: 1,
		Uid:   nfstypes.Uid3(0),
		Gid:   nfstypes.Gid3(0),
		Size:  nfstypes.Size3(0), // size of file
		Used:  nfstypes.Size3(0), // actual disk space used
		Rdev: nfstypes.Specdata3{
			Specdata1: nfstypes.Uint32(0),
			Specdata2: nfstypes.Uint32(0),
		},
		Fsid:   nfstypes.Uint64(0),
		Fileid: nfstypes.Fileid3(ip.Inum), // this is a unique id per file
		Atime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last accessed
		Mtime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last modified
		Ctime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last time attributes were changed, including writes
	}
}

func (ip *Inode) Encode() []byte {
	enc := marshal.NewEnc(common.INODESZ)
	enc.PutInt(ip.Gen)
	enc.PutInt(ip.Parent)
	enc.PutInts(ip.Contents)
	enc.PutBytes(ip.Names)
	return enc.Finish()
}

func Decode(buf *buf.Buf, inum common.Inum) *Inode {
	ip := new(Inode)
	dec := marshal.NewDec(buf.Data)
	ip.Inum = inum
	ip.Gen = dec.GetInt()
	ip.Parent = dec.GetInt()
	ip.Contents = dec.GetInts(NENTRIES)
	ip.Names = dec.GetBytes(NENTRIES)
	return ip
}

// func (ip *Inode) WriteInode(atxn *alloctxn.BufTxn) {
// 	if ip.Inum >= atxn.Super.NInode() {
// 		panic("WriteInode")
// 	}
// 	d := ip.Encode()
// 	atxn.Buftxn.OverWrite(atxn.Super.Inum2Addr(ip.Inum), d)
// 	util.DPrintf(1, "WriteInode %v\n", ip)
// }
// 
// func (ip *Inode) FreeInode(atxn *alloctxn.AllocTxn) {
// 	ip.Kind = NF3FREE
// 	ip.Gen = ip.Gen + 1
// 	ip.WriteInode(atxn)
// 	atxn.FreeINum(ip.Inum)
// }
