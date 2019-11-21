package goose_nfs

import (
	"github.com/tchajed/goose/machine"
)

type fh struct {
	ino uint64
	gen uint64
}

func (fh3 Nfs_fh3) makeFh() fh {
	i := machine.UInt64Get(fh3.Data[0:8])
	g := machine.UInt64Get(fh3.Data[8:])
	return fh{ino: i, gen: g}
}

func MkRootFh3() Nfs_fh3 {
	d := make([]byte, 16)
	machine.UInt64Put(d[0:8], 0)
	machine.UInt64Put(d[8:16], 0)
	return Nfs_fh3{Data: d}
}
