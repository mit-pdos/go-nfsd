package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

// XXX keep track whether buffer was modified so that we don't write
// it into log on commit.
type Buf struct {
	slot  *Cslot
	blk   disk.Block
	blkno uint64
}

func (buf *Buf) lock() {
	buf.slot.lock()
}

func (buf *Buf) unlock() {
	buf.slot.unlock()
}

type Txn struct {
	log  *Log
	bc   *Cache          // a cache of Buf's shared between transactions
	bufs map[uint64]*Buf // Locked bufs in use by this transaction
	fs   *FsSuper
	ic   *Cache
}

// Returns a locked buf
func (txn *Txn) load(slot *Cslot, a uint64) *Buf {
	slot.lock()
	if slot.obj == nil {
		// blk hasn't been read yet from disk; read it and put
		// the buf with the read blk in the cache slot.
		blk := disk.Read(a)
		buf := &Buf{slot: slot, blk: blk, blkno: a}
		slot.obj = buf
	}
	buf := slot.obj.(*Buf)
	return buf
}

// Release locks and cache slots, but pin buffers in cache until they
// have been installed.
// XXX support installing
func (txn *Txn) release() {
	log.Printf("release bufs")
	for _, buf := range txn.bufs {
		buf.unlock()
		txn.bc.freeSlot(buf.blkno, true)
	}
}

// XXX wait if cannot reserve space in log
func Begin(log *Log, cache *Cache, fs *FsSuper, ic *Cache) *Txn {
	txn := &Txn{
		log:  log,
		bc:   cache,
		bufs: make(map[uint64]*Buf),
		fs:   fs,
		ic:   ic,
	}
	return txn
}

func (txn *Txn) Read(addr uint64) disk.Block {
	v, ok := txn.bufs[addr]
	if ok {
		// this transaction already has the buf locked
		return v.blk
	} else {
		slot := txn.bc.lookupSlot(addr)
		if slot == nil {
			return nil
		}
		// load the slot and lock it
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

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) Commit(inodes []*Inode) bool {
	log.Printf("commit\n")

	// may free an inode so must be done before Append
	txn.putInodes(inodes)

	// commit all buffers used by this transaction
	bufs := new([]Buf)
	for _, buf := range txn.bufs {
		*bufs = append(*bufs, *buf)
	}
	ok := (*txn.log).Append(*bufs)

	// release the buffers used in this transaction
	txn.release()

	// unlock all inodes used in this transaction
	unlockInodes(inodes)

	return ok
}

func (txn *Txn) Abort(inodes []*Inode) {
	log.Printf("abort\n")
	txn.release()
}
