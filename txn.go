package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

// XXX keep track whether buffer was modified so that we don't write
// it into log on commit.
type Buf struct {
	mu    *sync.RWMutex
	blk   disk.Block
	blkno uint64
}

type Txn struct {
	log   *Log
	cache *Cache          // a cache of Buf's shared between transactions
	bufs  map[uint64]*Buf // Locked bufs in use by this transaction
}

// Returns a locked buf
func (txn *Txn) load(slot *Cslot, a uint64) *Buf {
	slot.mu.Lock()
	if slot.obj == nil {
		// blk hasn't been read yet from disk; read it and put
		// the buf with the read blk in the cache slot.
		blk := disk.Read(a)
		buf := &Buf{mu: new(sync.RWMutex), blk: blk, blkno: a}
		slot.obj = buf
	}
	buf := slot.obj.(*Buf)
	buf.mu.Lock()
	slot.mu.Unlock()
	return buf
}

// Release locks and cache slots, but pin buffers in cache until they
// have been installed.
// XXX support installing
func (txn *Txn) release() {
	log.Printf("release bufs")
	for _, buf := range txn.bufs {
		buf.mu.Unlock()
		txn.cache.freeSlot(buf.blkno, true)
	}
}

// XXX wait if cannot reserve space in log
func Begin(log *Log, cache *Cache) *Txn {
	txn := &Txn{
		log:   log,
		cache: cache,
		bufs:  make(map[uint64]*Buf),
	}
	return txn
}

func (txn *Txn) Read(addr uint64) disk.Block {
	v, ok := txn.bufs[addr]
	if ok {
		// this transaction already has the buf locked
		return v.blk
	} else {
		slot := txn.cache.lookupSlot(addr)
		if slot == nil {
			return nil
		}
		// load the slot with a locked block
		buf := txn.load(slot, addr)
		txn.bufs[addr] = buf
		return buf.blk
	}
}

func (txn *Txn) Write(addr uint64, blk disk.Block) bool {
	_, ok := txn.bufs[addr]
	if !ok {
		panic("Write: blind write")
	}
	// This transaction owns the locked block; update the block
	txn.bufs[addr].blk = blk
	return true
}

func (txn *Txn) Commit() bool {
	log.Printf("commit\n")
	bufs := new([]Buf)
	for _, buf := range txn.bufs {
		*bufs = append(*bufs, *buf)
	}
	ok := (*txn.log).Append(*bufs)
	txn.release()
	return ok
}

func (txn *Txn) Abort() {
	log.Printf("abort\n")
	txn.release()
}
