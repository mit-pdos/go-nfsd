package inode

import (
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/txn"
)

type FsState struct {
	Fs     *fs.FsSuper
	Txn    *txn.Txn
	Icache *cache.Cache
	Balloc *alloc.Alloc
	Ialloc *alloc.Alloc
}

func MkFsState(fs *fs.FsSuper, txn *txn.Txn, icache *cache.Cache, balloc *alloc.Alloc, ialloc *alloc.Alloc) *FsState {
	st := &FsState{
		Fs:     fs,
		Txn:    txn,
		Icache: icache,
		Balloc: balloc,
		Ialloc: ialloc,
	}
	return st
}
