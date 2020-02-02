package addrlock

import (
	"sync"

	"github.com/mit-pdos/goose-nfsd/txn"
)

type lockShard struct {
	mu      *sync.Mutex
	cond    *sync.Cond
	holders map[uint64]txn.TransId
}

func mkLockShard() *lockShard {
	mu := new(sync.Mutex)
	a := &lockShard{
		mu:      mu,
		cond:    sync.NewCond(mu),
		holders: make(map[uint64]txn.TransId),
	}
	return a
}

func (lmap *lockShard) acquire(addr uint64, id txn.TransId) {
	lmap.mu.Lock()
	for {
		_, held := lmap.holders[addr]
		if !held {
			lmap.holders[addr] = id
			break
		}

		lmap.cond.Wait()
	}
	lmap.mu.Unlock()
}

func (lmap *lockShard) release(addr uint64) {
	lmap.mu.Lock()
	delete(lmap.holders, addr)
	lmap.mu.Unlock()
	lmap.cond.Broadcast()
}

func (lmap *lockShard) isLocked(addr uint64, id txn.TransId) bool {
	lmap.mu.Lock()
	holder, held := lmap.holders[addr]
	lmap.mu.Unlock()
	if !held {
		return false
	}
	return holder == id
}

const NSHARD uint64 = 43

type LockMap struct {
	shards []*lockShard
}

func MkLockMap() *LockMap {
	shards := make([]*lockShard, NSHARD)
	for i := uint64(0); i < NSHARD; i++ {
		shards[i] = mkLockShard()
	}
	a := &LockMap{
		shards: shards,
	}
	return a
}

func (lmap *LockMap) Acquire(flataddr uint64, id txn.TransId) {
	shard := lmap.shards[flataddr%NSHARD]
	shard.acquire(flataddr, id)
}

func (lmap *LockMap) Release(flataddr uint64, id txn.TransId) {
	shard := lmap.shards[flataddr%NSHARD]
	shard.release(flataddr)
}

func (lmap *LockMap) IsLocked(flataddr uint64, id txn.TransId) bool {
	shard := lmap.shards[flataddr%NSHARD]
	return shard.isLocked(flataddr, id)
}
