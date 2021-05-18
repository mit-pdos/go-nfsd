package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/go-journal/buf"
	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/shrinker"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/go-journal/txn"
	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/goose-nfsd/util/stats"
)

type Nfs struct {
	fsstate  *fstxn.FsState
	shrinkst *shrinker.ShrinkerSt
	// support unstable writes
	Unstable bool
	// statistics
	stats [NUM_NFS_OPS]stats.Op
}

func MakeNfs(d disk.Disk) *Nfs {
	// run first so that disk is initialized before mkLog
	super := super.MkFsSuper(d)
	util.DPrintf(1, "Super: "+
		"Size %d NBlockBitmap %d NInodeBitmap %d Maxaddr %d\n",
		d.Size(),
		super.NBlockBitmap, super.NInodeBitmap, super.Maxaddr)

	txn := txn.MkTxn(d) // runs recovery

	i := readRootInode(super)
	if i.Kind == 0 { // make a new file system?
		makeFs(super)
	}

	st := fstxn.MkFsState(super, txn)
	nfs := &Nfs{
		fsstate:  st,
		shrinkst: shrinker.MkShrinkerSt(st),
		Unstable: true,
	}
	if i.Kind == 0 {
		nfs.makeRootDir()
	}
	return nfs
}

func (nfs *Nfs) ShutdownNfs() {
	util.DPrintf(1, "Shutdown\n")
	nfs.shrinkst.Shutdown()
	nfs.fsstate.Txn.Shutdown()
	util.DPrintf(1, "Shutdown done\n")
}

// Terminates shrinker thread immediately
func (nfs *Nfs) Crash() {
	util.DPrintf(0, "Crash: terminate shrinker\n")
	nfs.shrinkst.Crash()
	nfs.ShutdownNfs()
}

func (nfs *Nfs) makeRootDir() {
	op := fstxn.Begin(nfs.fsstate)
	ip := op.GetInodeInumFree(common.ROOTINUM)
	if ip == nil {
		panic("makeRootDir")
	}
	dir.MkRootDir(ip, op)
	ok := op.Commit()
	if !ok {
		panic("makeRootDir")
	}
}

// Make an empty file system
func makeFs(super *super.FsSuper) {
	util.DPrintf(1, "mkfs")

	root := inode.MkRootInode()
	util.DPrintf(1, "root %v\n", root)
	raddr := super.Inum2Addr(common.ROOTINUM)
	rootblk := root.Encode()
	rootbuf := buf.MkBuf(raddr, common.INODESZ*8, rootblk)
	rootbuf.WriteDirect(super.Disk)

	markAlloc(super, super.DataStart(), super.MaxBnum())
}

func markAlloc(super *super.FsSuper, n common.Bnum, m common.Bnum) {
	util.DPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m,
		super.NBlockBitmap*common.NBITBLOCK)
	if n >= common.Bnum(common.NBITBLOCK) ||
		m >= common.Bnum(common.NBITBLOCK*super.NBlockBitmap) ||
		m < n {
		panic("markAlloc: configuration makes no sense")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < uint64(n); bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] = blk[byte] | 1<<bit
	}
	super.Disk.Write(uint64(super.BitmapBlockStart()), blk)

	var blk1 = blk
	blkno := m/common.Bnum(common.NBITBLOCK) + super.BitmapBlockStart()
	if blkno > super.BitmapBlockStart() {
		blk1 = make(disk.Block, disk.BlockSize)
	}
	for bn := uint64(m) % common.NBITBLOCK; bn < common.NBITBLOCK; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk1[byte] = blk1[byte] | 1<<bit
	}
	super.Disk.Write(uint64(blkno), blk1)

	// mark inode 0 and 1 as allocated
	blk2 := make(disk.Block, disk.BlockSize)
	blk2[0] = blk2[0] | 1<<0
	blk2[0] = blk2[0] | 1<<1
	super.Disk.Write(uint64(super.BitmapInodeStart()), blk2)
}

func readRootInode(super *super.FsSuper) *inode.Inode {
	addr := super.Inum2Addr(common.ROOTINUM)
	blk := super.Disk.Read(uint64(addr.Blkno))
	buf := buf.MkBufLoad(addr, common.INODESZ*8, blk)
	i := inode.Decode(buf, common.ROOTINUM)
	return i
}
