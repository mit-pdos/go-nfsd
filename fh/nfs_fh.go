package fh

import (
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/tchajed/goose/machine"
)

type Fh struct {
	Ino fs.Inum
	Gen uint64
}

func MakeFh(fh3 nfstypes.Nfs_fh3) Fh {
	i := machine.UInt64Get(fh3.Data[0:8])
	g := machine.UInt64Get(fh3.Data[8:])
	return Fh{Ino: fs.Inum(i), Gen: g}
}

func (fh Fh) MakeFh3() nfstypes.Nfs_fh3 {
	fh3 := nfstypes.Nfs_fh3{Data: make([]byte, 16)}
	machine.UInt64Put(fh3.Data[0:8], uint64(fh.Ino))
	machine.UInt64Put(fh3.Data[8:], fh.Gen)
	return fh3
}

func MkRootFh3() nfstypes.Nfs_fh3 {
	d := make([]byte, 16)
	machine.UInt64Put(d[0:8], uint64(fs.ROOTINUM))
	machine.UInt64Put(d[8:16], 1)
	return nfstypes.Nfs_fh3{Data: d}
}

func Equal(h1 nfstypes.Nfs_fh3, h2 nfstypes.Nfs_fh3) bool {
	if len(h1.Data) != len(h2.Data) {
		return false
	}
	var equal = true
	for i, x := range h1.Data {
		if x != h2.Data[i] {
			equal = false
			break
		}
	}
	return equal
}
