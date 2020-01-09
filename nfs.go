package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
	"github.com/mit-pdos/goose-nfsd/wal"
)

const ICACHESZ uint64 = 20               // XXX resurrect icache
const BCACHESZ uint64 = fs.HDRADDRS + 10 // At least as big as log

type Nfs struct {
	txn      *txn.Txn
	fs       *fs.FsSuper
	balloc   *alloc.Alloc
	ialloc   *alloc.Alloc
	shrinker *inode.Shrinker
}

func MkNfs() *Nfs {
	super := fs.MkFsSuper() // run first so that disk is initialized before mkLog
	util.DPrintf(1, "Super: %v\n", super)

	l := wal.MkLog()
	if l == nil {
		panic("mkLog failed")
	}

	initFs(super)

	txn := txn.MkTxn(super)
	balloc := alloc.MkAlloc(super.BitmapBlockStart(), super.NBlockBitmap)
	ialloc := alloc.MkAlloc(super.BitmapInodeStart(), super.NInodeBitmap)
	nfs := &Nfs{
		txn:      txn,
		fs:       super,
		balloc:   balloc,
		ialloc:   ialloc,
		shrinker: inode.MkShrinker(super, txn, balloc, ialloc),
	}
	nfs.makeRootDir()
	return nfs
}

func (nfs *Nfs) makeRootDir() {
	op := fstxn.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
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
	nfs.txn.Shutdown()
}
