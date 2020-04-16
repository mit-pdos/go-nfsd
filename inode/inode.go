package inode

import (
	// "github.com/tchajed/goose/machine/disk"
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/util"
)

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
