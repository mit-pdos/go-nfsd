package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

const ICACHESZ uint64 = 20               // XXX resurrect icache
const BCACHESZ uint64 = fs.HDRADDRS + 10 // At least as big as log

type Nfs struct {
	fsstate  *fstxn.FsState
	shrinker *inode.Shrinker
}

func MkNfs() *Nfs {
	super := fs.MkFsSuper() // run first so that disk is initialized before mkLog
	util.DPrintf(1, "Super: %v\n", super)

	initFs(super)

	txn := txn.MkTxn(super)
	icache := cache.MkCache(ICACHESZ)
	balloc := alloc.MkAlloc(super.BitmapBlockStart(), super.NBlockBitmap)
	ialloc := alloc.MkAlloc(super.BitmapInodeStart(), super.NInodeBitmap)
	st := fstxn.MkFsState(super, txn, icache, balloc, ialloc)
	nfs := &Nfs{
		fsstate:  st,
		shrinker: inode.MkShrinker(st),
	}
	nfs.makeRootDir()
	return nfs
}

func (nfs *Nfs) makeRootDir() {
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInodeInum(op, fs.ROOTINUM)
	if ip == nil {
		panic("makeRootDir")
	}
	dir.MkRootDir(ip, op)
	ok := inode.Commit(op, []*inode.Inode{ip})
	if !ok {
		panic("makeRootDir")
	}
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.shrinker.Shutdown()
	nfs.fsstate.Txn.Shutdown()
}
