package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
)

type Buf struct {
	slot  *Cslot
	blk   disk.Block
	blkno uint64
	dirty bool // has this block been written to?
	meta  bool // does the block contain metadata?
	fh    Fh   // for non-meta blocks of fh in unstable writes
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
	bufs map[uint64]*Buf // locked bufs in use by this transaction
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

// Unqualified write is always written to log. Assumes transaction has the buf locked.
func (txn *Txn) Write(addr uint64, blk disk.Block) bool {
	_, ok := txn.bufs[addr]
	if !ok {
		panic("Write: blind write")
	}
	log.Printf("write %d\n", addr)
	txn.bufs[addr].meta = true
	txn.bufs[addr].dirty = true
	txn.bufs[addr].blk = blk
	return true
}

// Write of a data block.  Assumes transaction has the buf locked
func (txn *Txn) WriteData(addr uint64, blk disk.Block) bool {
	_, ok := txn.bufs[addr]
	if !ok {
		panic("Write: blind write")
	}
	txn.bufs[addr].meta = false
	txn.bufs[addr].dirty = true
	txn.bufs[addr].blk = blk
	return true
}

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) dirtyBufs() ([]*Buf, bool) {
	var meta bool = false
	bufs := new([]*Buf)
	for _, buf := range txn.bufs {
		if buf.dirty {
			*bufs = append(*bufs, buf)
		}
		if buf.meta {
			meta = true
		}
	}
	return *bufs, meta
}

func (txn *Txn) clearDirty(bufs []*Buf) {
	for _, b := range bufs {
		b.dirty = false
	}
}

func (txn *Txn) Commit(inodes []*Inode) bool {
	// may free an inode so must be done before Append
	txn.putInodes(inodes)

	// commit all buffers written by this transaction
	bufs, _ := txn.dirtyBufs()
	ok := (*txn.log).Append(bufs)
	txn.clearDirty(bufs)

	// release the buffers used in this transaction
	txn.release()

	// unlock all inodes used in this transaction
	unlockInodes(inodes)

	return ok
}

// XXX don't write inode if mtime is only change
func (txn *Txn) CommitData(inodes []*Inode, fh Fh) bool {
	return txn.Commit(inodes)
}

// If metadata changes, write metadata and data blocks through log,
// but without waiting for commit. otherwise, don't write data to log,
// just leave it in the cache.
func (txn *Txn) CommitUnstable(inodes []*Inode, fh Fh) bool {
	log.Printf("CommitUnstable\n")
	var ok bool = true
	var meta bool = false
	if len(inodes) > 1 {
		panic("CommitUnstable")
	}

	bufs, meta := txn.dirtyBufs()
	if meta {
		// append to in-memory log, but don't wait for the logger
		// to complete diskAppend
		log.Printf("Commitunstable: log\n")
		ok, _ = (*txn.log).MemAppend(bufs)
	} else {
		// don't write buffers, but tag them with fh for CommitFh
		for _, buf := range bufs {
			buf.fh = fh
		}
		// release will pin buffers in cache until CommmitFh
	}

	txn.release()

	unlockInodes(inodes)

	return ok
}

// Write data blocks associated with fh to their home location (i.e., not though log)
func (txn *Txn) CommitFh(fh Fh, inodes []*Inode) bool {
	bufs := txn.bc.BufsFh(fh)
	ids := new([]uint64)
	for _, b := range bufs {
		log.Printf("CommitFh: install blk %v\n", b.blkno)
		*ids = append(*ids, b.blkno)
		disk.Write(b.blkno, b.blk)
	}
	txn.clearDirty(bufs)
	txn.bc.Unpin(*ids)
	unlockInodes(inodes)
	return true
}

func (txn *Txn) Abort(inodes []*Inode) bool {
	log.Printf("abort\n")

	// An an abort may free an inode, which results in dirty
	// buffers that need to be written to log. So, call commit.
	return txn.Commit(inodes)
}
