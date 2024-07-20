package simple

import (
	"github.com/goose-lang/goose/machine/disk"
	"github.com/mit-pdos/go-journal/jrnl"

	"github.com/mit-pdos/go-journal/lockmap"
	"github.com/mit-pdos/go-journal/obj"
)

type Nfs struct {
	t *obj.Log
	l *lockmap.LockMap
}

func Mkfs(d disk.Disk) *obj.Log {
	log := obj.MkLog(d)
	op := jrnl.Begin(log)
	inodeInit(op)
	ok := op.CommitWait(true)
	if !ok {
		return nil
	}
	return log
}

func Recover(d disk.Disk) *Nfs {
	log := obj.MkLog(d) // runs recovery
	lockmap := lockmap.MkLockMap()

	nfs := &Nfs{
		t: log,
		l: lockmap,
	}
	return nfs
}

func MakeNfs(d disk.Disk) *Nfs {
	log := obj.MkLog(d) // runs recovery

	op := jrnl.Begin(log)
	inodeInit(op)
	ok := op.CommitWait(true)
	if !ok {
		return nil
	}

	lockmap := lockmap.MkLockMap()

	nfs := &Nfs{
		t: log,
		l: lockmap,
	}

	return nfs
}
