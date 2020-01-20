package inode

import (
	"sync"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/util"
)

type Shrinker struct {
	mu       *sync.Mutex
	condShut *sync.Cond
	nthread  uint32
	fsstate  *fstxn.FsState
}

var shrinker *Shrinker

func MkShrinker(st *fstxn.FsState) *Shrinker {
	mu := new(sync.Mutex)
	shrinker = &Shrinker{
		mu:       mu,
		condShut: sync.NewCond(mu),
		nthread:  0,
		fsstate:  st,
	}
	return shrinker
}

func (shrinker *Shrinker) Shutdown() {
	shrinker.mu.Lock()
	for shrinker.nthread > 0 {
		util.DPrintf(1, "ShutdownNfs: wait %d\n", shrinker.nthread)
		shrinker.condShut.Wait()
	}
	shrinker.mu.Unlock()
}

func singletonTrans(ip *Inode) []*Inode {
	return []*Inode{ip}
}

func shrink(inum fs.Inum, oldsz uint64) {
	var bn = util.RoundUp(oldsz, disk.BlockSize)
	util.DPrintf(1, "Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		op := fstxn.Begin(shrinker.fsstate)
		ip := getInodeInumFree(op, inum)
		if ip == nil {
			panic("shrink")
		}
		if ip.Size >= oldsz { // file has grown again or resize didn't commit
			ok := Commit(op, singletonTrans(ip))
			if !ok {
				panic("shrink")
			}
			break
		}
		cursz := util.RoundUp(ip.Size, disk.BlockSize)
		util.DPrintf(1, "shrink: bn %d cursz %d\n", bn, cursz)
		// 4: inode block, 2xbitmap block, indirect block, double indirect
		for bn > cursz && op.NumberDirty()+4 < op.LogSz() {
			bn = bn - 1
			if bn < NDIRECT {
				op.FreeBlock(ip.blks[bn])
				ip.blks[bn] = 0
			} else {
				var off = bn - NDIRECT
				if off < NBLKBLK {
					freeroot := ip.indshrink(op, ip.blks[INDIRECT], 1, off)
					if freeroot != 0 {
						op.FreeBlock(ip.blks[INDIRECT])
						ip.blks[INDIRECT] = 0
					}
				} else {
					off = off - NBLKBLK
					freeroot := ip.indshrink(op, ip.blks[DINDIRECT], 2, off)
					if freeroot != 0 {
						op.FreeBlock(ip.blks[DINDIRECT])
						ip.blks[DINDIRECT] = 0
					}
				}
			}
		}
		ip.WriteInode(op)
		ok := Commit(op, singletonTrans(ip))
		if !ok {
			panic("shrink")
		}
		if bn <= cursz {
			break
		}
	}
	util.DPrintf(1, "Shrinker: done shrinking %d to bn %d\n", inum, bn)
	shrinker.mu.Lock()
	shrinker.nthread = shrinker.nthread - 1
	shrinker.condShut.Signal()
	shrinker.mu.Unlock()
}
