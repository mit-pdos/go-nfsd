package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/txn"
)

func MakeNfs(d disk.Disk) *Nfs {
	txn := txn.MkTxn(d) // runs recovery

	btxn := buftxn.Begin(txn)
	inodeInit(btxn)
	ok := btxn.CommitWait(true)
	if !ok {
		return nil
	}

	lockmap := lockmap.MkLockMap()

	// XXX mkfs needs to happen somewhere

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}

	return nfs
}
