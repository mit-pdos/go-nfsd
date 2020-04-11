package goose_nfs

import (
	"fmt"
	"path/filepath"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"

	"math/rand"
	"os"
	"strconv"
)

type Nfs struct {
	Name     *string
	glocks   *lockmap.LockMap // for now, we only use block 0
	fs       *super.FsSuper
	txn      *txn.Txn
	bitmap   []byte
}

func MkNfsMem(sz uint64) *Nfs {
	return MakeNfs(nil, sz)
}

func MkNfsName(name string, sz uint64) *Nfs {
	return MakeNfs(&name, sz)
}

func MkNfs(sz uint64) *Nfs {
	r := rand.Uint64()
	tmpdir := "/dev/shm"
	f, err := os.Stat(tmpdir)
	if !(err == nil && f.IsDir()) {
		tmpdir = os.TempDir()
	}
	n := filepath.Join(tmpdir, "goose"+strconv.FormatUint(r, 16)+".img")
	name := &n
	return MakeNfs(name, sz)
}

func MakeNfs(name *string, sz uint64) *Nfs {
	var d disk.Disk
	if name == nil {
		d = disk.NewMemDisk(sz)
	} else {
		util.DPrintf(1, "MakeNfs: creating file disk at %s", *name)
		var err error
		d, err = disk.NewFileDisk(*name, sz)
		if err != nil {
			panic(fmt.Errorf("could not create file disk: %v", err))
		}
	}
	// run first so that disk is initialized before mkLog
	super := super.MkFsSuper(d)
	util.DPrintf(1, "Super: sz %d %v\n", sz, super)

	txn := txn.MkTxn(super) // runs recovery

	makeFs(super)

	nfs := &Nfs{
		Name:     name,
		glocks:   lockmap.MkLockMap(),
		fs:       super,
		txn:      txn,
	}
	nfs.makeRootDir()
	nfs.readBitmap()
	return nfs
}

func (nfs *Nfs) readBitmap() {
	var bitmap []byte
	start := nfs.fs.BitmapInodeStart()
	for i := uint64(0); i < nfs.fs.NInodeBitmap; i++ {
		blk := nfs.fs.Disk.Read(uint64(start) + i)
		bitmap = append(bitmap, blk...)
	}
	nfs.bitmap = bitmap
}

func (nfs *Nfs) doShutdown(destroy bool) {
	util.DPrintf(1, "Shutdown %v\n", destroy)
	nfs.txn.Shutdown()

	if destroy && nfs.Name != nil {
		util.DPrintf(1, "Destroy %v\n", *nfs.Name)
		err := os.Remove(*nfs.Name)
		if err != nil {
			panic(err)
		}
	}

	util.DPrintf(1, "Shutdown done\n")
}

func (nfs *Nfs) ShutdownNfsDestroy() {
	nfs.doShutdown(true)
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.doShutdown(false)
}

// Terminates shrinker thread immediately
func (nfs *Nfs) Crash() {
	util.DPrintf(0, "Crash: terminate shrinker\n")
	nfs.ShutdownNfs()
}

func (nfs *Nfs) makeRootDir() {
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
	// util.DPrintf(1, "inode block: %v\n", super.Disk.Read(uint64(raddr.Blkno))[raddr.Off/8:raddr.Off/8+common.INODESZ])

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
