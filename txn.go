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
	blk   *disk.Block
	blkno uint64
}

type Txn struct {
	log   *Log
	cache *Cache
	bufs  map[uint64]*Buf
}

// Returns a locked buf
func (txn *Txn) load(co *Cobj, a uint64) *Buf {
	var blk *disk.Block
	co.mu.Lock()
	if !co.valid {
		log.Printf("load block %d\n", a)
		blk := disk.Read(a)
		co.obj = &blk
		co.valid = true
	}
	blk = co.obj.(*disk.Block)
	buf := &Buf{mu: new(sync.RWMutex), blk: blk, blkno: a}
	buf.mu.Lock()
	co.mu.Unlock()
	return buf
}

// Release locks
func (txn *Txn) release() {
	log.Printf("release bufs")
	for _, buf := range txn.bufs {
		buf.mu.Unlock()
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

func (txn *Txn) Write(addr uint64, blk *disk.Block) bool {
	var ret bool = true
	_, ok := txn.bufs[addr]
	if ok {
		txn.bufs[addr].blk = blk
	}
	if !ok {
		if addr == LOGMAXBLK {
			// TODO: should be able to return early here
			ret = false
		} else {
			panic("lock buf first")
			txn.bufs[addr].blk = blk
		}
	}
	return ret
}

func (txn *Txn) Read(addr uint64) *disk.Block {
	v, ok := txn.bufs[addr]
	if ok {
		return v.blk
	} else {
		a := addr + LOGEND
		co := txn.cache.getputObj(addr + LOGEND)
		if co == nil {
			return nil
		}
		buf := txn.load(co, a)
		txn.bufs[addr] = buf
		return buf.blk
	}
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
	txn.release()
}
