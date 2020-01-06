package goose_nfs

import (
	"sync"
)

type alock struct {
	txn *txn
}

type lockMap struct {
	mu    *sync.Mutex
	addrs *addrMap
}

func mkLockMap() *lockMap {
	a := &lockMap{
		mu:    new(sync.Mutex),
		addrs: mkAddrMap(),
	}
	return a
}

func (lmap *lockMap) isLocked(addr addr, txn *txn) bool {
	locked := false
	lmap.mu.Lock()
	e := lmap.addrs.lookup(addr)
	if e != nil {
		l := e.(*alock)
		if l.txn == txn {
			locked = true
		}
	}
	lmap.mu.Unlock()
	return locked
}

// atomically lookup and add addr
func (lmap *lockMap) lookupadd(addr addr, txn *txn) bool {
	lmap.mu.Lock()
	e := lmap.addrs.lookup(addr)
	if e == nil {
		alock := &alock{txn: txn}
		lmap.addrs.insert(addr, alock)
		lmap.mu.Unlock()
		return true
	}
	dPrintf(5, "LookupAdd already locked %v %v\n", addr, e)
	lmap.mu.Unlock()
	return false
}

func (lmap *lockMap) acquire(addr addr, txn *txn) {
	for {
		if lmap.lookupadd(addr, txn) {
			break
		}
		// XXX condition variable?
		continue

	}
	dPrintf(5, "%p: acquire: %v\n", txn, addr)
}

func (lmap *lockMap) dorelease(addr addr, txn *txn) {
	dPrintf(5, "%p: release: %v\n", txn, addr)
	e := lmap.addrs.lookup(addr)
	if e == nil {
		panic("release")
	}
	alock := e.(*alock)
	if alock.txn != txn {
		panic("release")
	}
	lmap.addrs.del(addr)
}

func (lmap *lockMap) release(addr addr, txn *txn) {
	lmap.mu.Lock()
	lmap.dorelease(addr, txn)
	lmap.mu.Unlock()
}

// release all blocks held by txn
func (lmap *lockMap) releaseTxn(txn *txn) {
	lmap.mu.Lock()
	var addrs = make([]addr, 0)
	lmap.addrs.apply(func(a addr, e interface{}) {
		alock := e.(*alock)
		if alock.txn == txn {
			addrs = append(addrs, a)
		}
	})
	for _, a := range addrs {
		lmap.dorelease(a, txn)
	}
	lmap.mu.Unlock()
}
