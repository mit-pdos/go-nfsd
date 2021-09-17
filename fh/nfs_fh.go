package fh

import (
	"github.com/goose-lang/std"
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-nfsd/nfstypes"
)

type Fh struct {
	Ino common.Inum
	Gen uint64
}

func MakeFh(fh3 nfstypes.Nfs_fh3) Fh {
	dec := marshal.NewDec(fh3.Data)
	i := dec.GetInt()
	g := dec.GetInt()
	return Fh{Ino: common.Inum(i), Gen: g}
}

func (fh Fh) MakeFh3() nfstypes.Nfs_fh3 {
	enc := marshal.NewEnc(16)
	enc.PutInt(uint64(fh.Ino))
	enc.PutInt(uint64(fh.Gen))
	fh3 := nfstypes.Nfs_fh3{Data: enc.Finish()}
	return fh3
}

func MkRootFh3() nfstypes.Nfs_fh3 {
	enc := marshal.NewEnc(16)
	enc.PutInt(uint64(common.ROOTINUM))
	enc.PutInt(uint64(1))
	return nfstypes.Nfs_fh3{Data: enc.Finish()}
}

func Equal(h1 nfstypes.Nfs_fh3, h2 nfstypes.Nfs_fh3) bool {
	return std.BytesEqual(h1.Data, h2.Data)
}
