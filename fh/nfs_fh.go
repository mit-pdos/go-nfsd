package fh

import (
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
	if len(h1.Data) != len(h2.Data) {
		return false
	}
	var equal = true
	for i, x := range h1.Data {
		if x != h2.Data[i] {
			equal = false
			// break FIXME not supported by Goose
		}
	}
	return equal
}
