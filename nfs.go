package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/alloc"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"

	"math/rand"
	"os"
	"strconv"
)

const ICACHESZ uint64 = 20

type Nfs struct {
	Name     *string
	fsstate  *fstxn.FsState
	shrinker *inode.Shrinker
}

func MkNfsMem() *Nfs {
	return MakeNfs(nil)
}

func MkNfsName(name string) *Nfs {
	return MakeNfs(&name)
}

func MkNfs() *Nfs {
	r := rand.Uint64()
	n := "/dev/shm/goose" + strconv.FormatUint(r, 16) + ".img"
	name := &n
	return MakeNfs(name)
}

func MakeNfs(name *string) *Nfs {
	sz := uint64(100 * 1000)
	// run first so that disk is initialized before mkLog
	super := fs.MkFsSuper(sz, name)
	util.DPrintf(1, "Super: %v\n", super)

	txn := txn.MkTxn(super)
	icache := cache.MkCache(ICACHESZ)
	balloc := alloc.MkAlloc(super.BitmapBlockStart(), super.NBlockBitmap)
	ialloc := alloc.MkAlloc(super.BitmapInodeStart(), super.NInodeBitmap)
	st := fstxn.MkFsState(super, txn, icache, balloc, ialloc)
	nfs := &Nfs{
		Name:     name,
		fsstate:  st,
		shrinker: inode.MkShrinker(st),
	}
	i := ReadRootInode(super)
	if i.Kind == 0 {
		makeFs(super)
		nfs.makeRootDir()
	}
	return nfs
}

func (nfs *Nfs) doShutdown(destroy bool) {
	util.DPrintf(1, "Shutdown %v\n", destroy)
	nfs.shrinker.Shutdown()
	nfs.fsstate.Txn.Shutdown()

	if destroy {
		os.Remove(*nfs.Name)
	}

	util.DPrintf(1, "Shutdown done\n")
}

func (nfs *Nfs) ShutdownNfsDestroy() {
	nfs.doShutdown(true)
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.doShutdown(false)
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

// Make an empty file system
func makeFs(super *fs.FsSuper) {
	// inum = 0 is reserved
	nulli := inode.MkNullInode()
	naddr := super.Inum2Addr(fs.NULLINUM)
	d := nulli.Encode()
	b := buf.MkBuf(naddr, d)
	b.WriteDirect(super.Disk)

	root := inode.MkRootInode()
	util.DPrintf(1, "root %v\n", root)
	raddr := super.Inum2Addr(fs.ROOTINUM)
	rootblk := root.Encode()
	rootbuf := buf.MkBuf(raddr, rootblk)
	rootbuf.WriteDirect(super.Disk)

	markAlloc(super, super.DataStart(), super.Maxaddr)
}

func markAlloc(super *fs.FsSuper, n uint64, m uint64) {
	util.DPrintf(1, "markAlloc: [0, %d) and [%d,%d)\n", n, m,
		super.NBlockBitmap*alloc.NBITBLOCK)
	if n >= alloc.NBITBLOCK || m >= alloc.NBITBLOCK*super.NBlockBitmap ||
		m < alloc.NBITBLOCK {
		panic("markAlloc: configuration makes no sense")
	}
	blk := make(disk.Block, disk.BlockSize)
	for bn := uint64(0); bn < n; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk[byte] = blk[byte] | 1<<bit
	}
	super.Disk.Write(super.BitmapBlockStart(), blk)

	blk1 := make(disk.Block, disk.BlockSize)
	blkno := m/alloc.NBITBLOCK + super.BitmapBlockStart()
	for bn := m % disk.BlockSize; bn < alloc.NBITBLOCK; bn++ {
		byte := bn / 8
		bit := bn % 8
		blk1[byte] = blk1[byte] | 1<<bit
	}
	super.Disk.Write(blkno, blk1)

	// mark inode 0 and 1 as allocated
	blk2 := make(disk.Block, disk.BlockSize)
	blk2[0] = blk2[0] | 1<<0
	blk2[0] = blk2[0] | 1<<1
	super.Disk.Write(super.BitmapInodeStart(), blk2)
}

// For boot up
func ReadRootInode(super *fs.FsSuper) *inode.Inode {
	addr := super.Inum2Addr(fs.ROOTINUM)
	blk := super.Disk.Read(addr.Blkno)
	buf := buf.MkBufLoad(addr, blk)
	i := inode.Decode(buf, fs.ROOTINUM)
	return i
}
