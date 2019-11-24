package goose_nfs

import (
	"github.com/tchajed/goose/machine"
)

type Fh struct {
	ino uint64
	gen uint64
}

func (fh3 Nfs_fh3) makeFh() Fh {
	i := machine.UInt64Get(fh3.Data[0:8])
	g := machine.UInt64Get(fh3.Data[8:])
	return Fh{ino: i, gen: g}
}

func (fh Fh) makeFh3() Nfs_fh3 {
	fh3 := Nfs_fh3{Data: make([]byte, 16)}
	machine.UInt64Put(fh3.Data[0:8], fh.ino)
	machine.UInt64Put(fh3.Data[8:], fh.gen)
	return fh3
}

func MkRootFh3() Nfs_fh3 {
	d := make([]byte, 16)
	machine.UInt64Put(d[0:8], ROOTINUM)
	machine.UInt64Put(d[8:16], 0)
	return Nfs_fh3{Data: d}
}
