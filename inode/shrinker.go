package inode

import (
	"sync"

	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/util"
)

//
// Freeing of a file.  If file is large (i.e,., has indirect blocks),
// freeing is done in separate thread, using perhaps multiple
// transactions to ensure that free fits in the log.
//

type ShrinkerSt struct {
	mu       *sync.Mutex
	condShut *sync.Cond
	nthread  uint32
	fsstate  *fstxn.FsState
}

var shrinkst *ShrinkerSt

func MkShrinkerSt(st *fstxn.FsState) *ShrinkerSt {
	mu := new(sync.Mutex)
	shrinkst = &ShrinkerSt{
		mu:       mu,
		condShut: sync.NewCond(mu),
		nthread:  0,
		fsstate:  st,
	}
	return shrinkst
}

func (shrinker *ShrinkerSt) Shutdown() {
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

// 5: inode block, 2xbitmap block, indirect block, double indirect
func enoughLogSpace(op *fstxn.FsTxn) bool {
	return op.NumberDirty()+5 < op.LogSz()
}

func (ip *Inode) smallFileFits(op *fstxn.FsTxn) bool {
	return ip.blks[INDIRECT] == 0 && op.NumberDirty()+NDIRECT+2 < op.LogSz()
}

func (ip *Inode) shrink(op *fstxn.FsTxn, bn uint64) uint64 {
	cursz := util.RoundUp(ip.Size, disk.BlockSize)
	util.DPrintf(1, "shrink: bn %d cursz %d\n", bn, cursz)
	for bn > cursz && enoughLogSpace(op) {
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
	return bn
}

func shrinker(inum fs.Inum, oldsz uint64) {
	var bn = util.RoundUp(oldsz, disk.BlockSize)
	util.DPrintf(1, "Shrinker: shrink %d from bn %d\n", inum, bn)
	for {
		op := fstxn.Begin(shrinkst.fsstate)
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
		bn = ip.shrink(op, bn)
		ok := Commit(op, singletonTrans(ip))
		if !ok {
			panic("shrink")
		}
		if bn <= cursz {
			break
		}
	}
	util.DPrintf(1, "Shrinker: done shrinking %d to bn %d\n", inum, bn)
	shrinkst.mu.Lock()
	shrinkst.nthread = shrinkst.nthread - 1
	shrinkst.condShut.Signal()
	shrinkst.mu.Unlock()
}

// Frees indirect bn.  Assumes if bn is cleared, then all blocks > bn
// have been cleared
func (ip *Inode) indshrink(op *fstxn.FsTxn, root uint64, level uint64, bn uint64) uint64 {
	if level == 0 {
		return root
	}
	divisor := pow(level - 1)
	off := (bn / divisor)
	ind := bn % divisor
	boff := off * 8
	buf := op.ReadBlock(root)
	nxtroot := machine.UInt64Get(buf.Blk[boff : boff+8])
	if nxtroot != 0 {
		freeroot := ip.indshrink(op, nxtroot, level-1, ind)
		if freeroot != 0 {
			machine.UInt64Put(buf.Blk[boff:boff+8], 0)
			buf.SetDirty()
			op.FreeBlock(freeroot)
		}
	}
	if off == 0 && ind == 0 {
		return root
	} else {
		return 0
	}
}
