package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"
)

type Txn struct {
	log   *Log
	cache *Cache
	blks  map[uint64]*disk.Block
}

func (txn *Txn) load(co *Cobj, a uint64) *disk.Block {
	var blk *disk.Block
	co.mu.Lock()
	if !co.valid {
		blk := disk.Read(a)
		co.obj = &blk
		co.valid = true
	}
	blk = co.obj.(*disk.Block)
	co.mu.Unlock()
	return blk
}

// XXX wait if cannot reserve space in log
func Begin(log *Log, cache *Cache) *Txn {
	txn := &Txn{
		log:   log,
		cache: cache,
		blks:  make(map[uint64]*disk.Block),
	}
	return txn
}

func (txn *Txn) Write(addr uint64, buf *disk.Block) bool {
	var ret bool = true
	_, ok := txn.blks[addr]
	if ok {
		txn.blks[addr] = buf
	}
	if !ok {
		if addr == LOGMAXBLK {
			// TODO: should be able to return early here
			ret = false
		} else {
			txn.blks[addr] = buf
		}
	}
	return ret
}

func (txn *Txn) Read(addr uint64) *disk.Block {
	v, ok := txn.blks[addr]
	if ok {
		return v
	} else {
		a := addr + LOGEND
		co := txn.cache.getputObj(addr + LOGEND)
		if co == nil {
			return nil
		}
		blk := txn.load(co, a)
		return blk
	}
}

func (txn *Txn) Commit() bool {
	blks := new([]disk.Block)
	for _, v := range txn.blks {
		*blks = append(*blks, *v)
	}
	ok := (*txn.log).Append(*blks)
	return ok
}

func (txn *Txn) Abort() {
}
