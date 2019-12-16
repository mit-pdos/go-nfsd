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

func (fh fh) makeFh3() Nfs_fh3 {
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

func (fh3 Nfs_fh3) equal(h Nfs_fh3) bool {
	if len(fh3.Data) != len(h.Data) {
		return false
	}
	for i, x := range fh3.Data {
		if x != h.Data[i] {
			return false
		}
	}
	return true
}
