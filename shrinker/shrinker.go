package shrinker

import (
	"sync"

	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/util"
)

func shrinker(inum common.Inum) {
	var more = true
	for more {
		op := fstxn.Begin(shrinkst.fsstate)
		ip := op.GetInodeInumFree(inum)
		if ip == nil {
			panic("shrink")
		}
		more = ip.Shrink(op.Atxn)
		ok := op.Commit()
		if !ok {
			panic("shrink")
		}
	}
	util.DPrintf(1, "Shrinker: done shrinking # %d\n", inum)
	shrinkst.mu.Lock()
	shrinkst.nthread = shrinkst.nthread - 1
	shrinkst.condShut.Signal()
	shrinkst.mu.Unlock()
}

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

// for large files, start a separate thread
func StartShrinker(inum common.Inum) {
	util.DPrintf(1, "start shrink thread\n")
	shrinkst.mu.Lock()
	shrinkst.nthread = shrinkst.nthread + 1
	shrinkst.mu.Unlock()
	go func() { shrinker(inum) }()
}

// If caller changes file size and shrinking is in progress (because
// an earlier call truncated the file), then help/wait with/for
// shrinking.
func HelpShrink(op *fstxn.FsTxn, ip *inode.Inode) (*fstxn.FsTxn, bool) {
	var ok bool = true
	inum := ip.Inum
	for ip.IsShrinking() {
		util.DPrintf(1, "%d: HelpShrink %v\n", op.Atxn.Id(), ip.Inum)
		more := ip.Shrink(op.Atxn)
		ok = op.Commit()
		op = fstxn.Begin(op.Fs)
		if !more || !ok {
			break
		}
		ip = op.GetInodeLocked(inum)
	}
	return op, ok
}
