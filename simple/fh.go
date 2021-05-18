package simple

import (
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

type Fh struct {
	Ino common.Inum
}

func MakeFh(fh3 nfstypes.Nfs_fh3) Fh {
	dec := marshal.NewDec(fh3.Data)
	i := dec.GetInt()
	return Fh{Ino: common.Inum(i)}
}

func (fh Fh) MakeFh3() nfstypes.Nfs_fh3 {
	enc := marshal.NewEnc(16)
	enc.PutInt(uint64(fh.Ino))
	fh3 := nfstypes.Nfs_fh3{Data: enc.Finish()}
	return fh3
}

func MkRootFh3() nfstypes.Nfs_fh3 {
	enc := marshal.NewEnc(16)
	enc.PutInt(uint64(common.ROOTINUM))
	return nfstypes.Nfs_fh3{Data: enc.Finish()}
}
