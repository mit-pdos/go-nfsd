package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

func MakeNfs(d disk.Disk) *Nfs {
	// run first so that disk is initialized before mkLog
	super := super.MkFsSuper(d)
	util.DPrintf(1, "Super: %v\n", super)

	txn := txn.MkTxn(d) // runs recovery

	lockmap := lockmap.MkLockMap()

	// XXX mkfs needs to happen somewhere

	nfs := &Nfs{
		t: txn,
		s: super,
		l: lockmap,
	}

	return nfs
}
