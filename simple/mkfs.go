package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/twophase"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

type Nfs struct {
	t *txn.Txn
	l *lockmap.LockMap
}

func Mkfs(d disk.Disk) *txn.Txn {
	txn := txn.MkTxn(d)
	l := lockmap.MkLockMap()
	tptxn := twophase.Begin(txn, l)
	inodeInit(tptxn)
	ok := tptxn.Commit()
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
	lockmap := lockmap.MkLockMap()

	tptxn := twophase.Begin(txn, lockmap)
	inodeInit(tptxn)
	ok := tptxn.Commit()
	if !ok {
		return nil
	}

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}

	return nfs
}
