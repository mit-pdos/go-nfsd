package fstxn

import (
	"github.com/mit-pdos/goose-nfsd/alloctxn"
	"github.com/mit-pdos/goose-nfsd/cache"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// fstxn implements transactions using alloctxn.  It adds to alloctxn
// support for locking inodes and an inode cache.
//

type FsTxn struct {
	Fs     *FsState
	Atxn   *alloctxn.AllocTxn
	inodes map[common.Inum]*inode.Inode
}

func Begin(fsstate *FsState) *FsTxn {
	op := &FsTxn{
		Fs: fsstate,
		Atxn: alloctxn.Begin(fsstate.Super, fsstate.Txn, fsstate.Balloc,
			fsstate.Ialloc),
		inodes: make(map[common.Inum]*inode.Inode),
	}
	return op
}

func (op *FsTxn) addInode(ip *inode.Inode) {
	op.inodes[ip.Inum] = ip
}

func (op *FsTxn) lookupInode(inum common.Inum) *inode.Inode {
	i := op.inodes[inum]
	return i
}

func (op *FsTxn) OwnInum(inum common.Inum) bool {
	_, ok := op.inodes[inum]
	return ok
}

func (op *FsTxn) doneInode(ip *inode.Inode) {
	delete(op.inodes, ip.Inum)
}

func (op *FsTxn) releaseInodes() {
	for _, ip := range op.inodes {
		op.ReleaseInode(ip)
	}
}

func (op *FsTxn) AllocInode(kind nfstypes.Ftype3) *inode.Inode {
	var ip *inode.Inode
	inum := op.Atxn.AllocINum()
	if inum != common.NULLINUM {
		ip = op.GetInodeLocked(inum)
		if ip.Kind != inode.NF3FREE {
			panic("AllocInode")
		}
		if !ip.IsShrinking() {
			util.DPrintf(1, "AllocInode -> # %v\n", inum)
			ip.InitInode(inum, kind)
			ip.WriteInode(op.Atxn)
		}
	}
	return ip
}

func (op *FsTxn) ReleaseInode(ip *inode.Inode) {
	util.DPrintf(1, "ReleaseInode %v\n", ip)
	op.doneInode(ip)
	op.Fs.Lockmap.Release(ip.Inum, op.Atxn.Id())
}

func (op *FsTxn) LockInode(inum common.Inum) *cache.Cslot {
	op.Fs.Lockmap.Acquire(inum, op.Atxn.Id())
	cslot := op.Fs.Icache.LookupSlot(uint64(inum))
	if cslot == nil {
		panic("GetInodeLocked")
	}
	return cslot
}

func (op *FsTxn) GetInodeLocked(inum common.Inum) *inode.Inode {
	cslot := op.LockInode(inum)
	if cslot.Obj == nil {
		addr := op.Fs.Super.Inum2Addr(inum)
		buf := op.Atxn.Buftxn.ReadBuf(addr)
		i := inode.Decode(buf, inum)
		util.DPrintf(1, "GetInodeLocked # %v: read inode from disk\n", inum)
		cslot.Obj = i
	}
	ip := cslot.Obj.(*inode.Inode)
	op.addInode(ip)
	util.DPrintf(1, "%d: GetInodeLocked %v\n", op.Atxn.Id(), ip)
	return ip
}

func (op *FsTxn) GetInodeInumFree(inum common.Inum) *inode.Inode {
	ip := op.GetInodeLocked(inum)
	return ip
}

func (op *FsTxn) GetInodeInum(inum common.Inum) *inode.Inode {
	ip := op.GetInodeInumFree(inum)
	if ip == nil {
		return nil
	}
	if ip.Kind == inode.NF3FREE {
		op.ReleaseInode(ip)
		return nil
	}
	if ip.Nlink == 0 {
		panic("getInodeInum")
	}
	return ip
}

func (op *FsTxn) GetInodeFh(fh3 nfstypes.Nfs_fh3) *inode.Inode {
	fh := fh.MakeFh(fh3)
	ip := op.GetInodeInum(fh.Ino)
	if ip == nil {
		return nil
	}
	if ip.Gen != fh.Gen {
		op.ReleaseInode(ip)
		return nil
	}
	return ip
}

// Assumes caller already has inode locked
func (op *FsTxn) GetInodeUnlocked(inum common.Inum) *inode.Inode {
	ip := op.lookupInode(inum)
	if ip == nil {
		panic("GetInodeUnlocked")
	}
	return ip
}
