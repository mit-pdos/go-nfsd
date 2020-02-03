package inode

import (
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
)

type FsState struct {
	Super  *super.FsSuper
	Txn    *txn.Txn
	Icache *cache.Cache
	Balloc *alloc.Alloc
	Ialloc *alloc.Alloc
}

func MkFsState(super *super.FsSuper, txn *txn.Txn, icache *cache.Cache, balloc *alloc.Alloc, ialloc *alloc.Alloc) *FsState {
	st := &FsState{
		Super:  super,
		Txn:    txn,
		Icache: icache,
		Balloc: balloc,
		Ialloc: ialloc,
	}
	return st
}
