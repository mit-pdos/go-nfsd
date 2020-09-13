package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/txn"
)

func MakeNfs(d disk.Disk) *Nfs {
	// run first so that disk is initialized before mkLog
	txn := txn.MkTxn(d) // runs recovery

	lockmap := lockmap.MkLockMap()

	// XXX mkfs needs to happen somewhere

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}

	return nfs
}
