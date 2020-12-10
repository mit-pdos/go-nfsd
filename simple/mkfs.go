package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/txn"
)

type Nfs struct {
	t *txn.Txn
	l *lockmap.LockMap
}

func Mkfs(d disk.Disk) *txn.Txn {
	txn := txn.MkTxn(d)
	btxn := buftxn.Begin(txn)
	inodeInit(btxn)
	ok := btxn.CommitWait(true)
	if !ok {
		return nil
	}
	return txn
}

func Recover(d disk.Disk) *Nfs {
	txn := txn.MkTxn(d) // runs recovery
	lockmap := lockmap.MkLockMap()

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}
	return nfs
}

func MakeNfs(d disk.Disk) *Nfs {
	txn := txn.MkTxn(d) // runs recovery

	btxn := buftxn.Begin(txn)
	inodeInit(btxn)
	ok := btxn.CommitWait(true)
	if !ok {
		return nil
	}

	lockmap := lockmap.MkLockMap()

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}

	return nfs
}
