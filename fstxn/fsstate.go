package fstxn

import (
	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util/stats"
)

const ICACHESZ uint64 = 100

type FsState struct {
	Super   *super.FsSuper
	Txn     *txn.Txn
	Icache  *cache.Cache
	Lockmap *lockmap.LockMap
	Balloc  *alloc.Alloc
	Ialloc  *alloc.Alloc
	Stats   [5]stats.Op
}

func readBitmap(super *super.FsSuper, start common.Bnum, len uint64) []byte {
	var bitmap []byte
	for i := uint64(0); i < len; i++ {
		blk := super.Disk.Read(uint64(start) + i)
		bitmap = append(bitmap, blk...)
	}
	return bitmap
}

func MkFsState(super *super.FsSuper, txn *txn.Txn) *FsState {
	balloc := alloc.MkAlloc(readBitmap(super, super.BitmapBlockStart(),
		super.NBlockBitmap))
	ialloc := alloc.MkAlloc(readBitmap(super, super.BitmapInodeStart(),
		super.NInodeBitmap))
	icache := cache.MkCache(ICACHESZ)
	st := &FsState{
		Super:   super,
		Txn:     txn,
		Icache:  icache,
		Lockmap: lockmap.MkLockMap(),
		Balloc:  balloc,
		Ialloc:  ialloc,
	}
	return st
}
