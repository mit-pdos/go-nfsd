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

// Release locks and cache slots
func (txn *Txn) release() {
	log.Printf("release bufs")
	for _, buf := range txn.bufs {
		buf.unlock()
		txn.bc.freeSlot(buf.blkno)
	}
}

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

// If Read cannot find a cache slot, wait until installer unpins
// blocks from cache: flush memlog, which may contain unstable writes,
// and signal installer.
func (txn *Txn) Read(addr uint64) disk.Block {
	b, ok := txn.bufs[addr]
	if ok {
		// this transaction already has the buf locked
		return b.blk
	} else {
		var slot *Cslot
		slot = txn.bc.lookupSlot(addr)
		for slot == nil {
			log.Printf("Read: WaitFlushMemLog and signal installer\n")
			txn.log.WaitFlushMemLog()
			txn.log.SignalInstaller()
			// Try again; a slot should free up eventually.
			slot = txn.bc.lookupSlot(addr)
		}
		// load the slot and lock it
		buf := txn.load(slot, addr)
		txn.bufs[addr] = buf
		return buf.blk
	}
}

// Release a not-used buffer during the transaction (e.g., during
// scanning inode or bitmap blocks that don't have free inodes or
// bits).
func (txn *Txn) ReleaseBlock(addr uint64) {
	b, ok := txn.bufs[addr]
	if !ok {
		log.Printf("ReleaseBlock: not present")
		return
	}
	if b.dirty {
		panic("ReleaseBlock")
	}
	b.unlock()
	txn.bc.freeSlot(b.blkno)
	delete(txn.bufs, addr)
}

// Unqualified write is always written to log. Assumes transaction has the buf locked.
func (txn *Txn) Write(addr uint64, blk disk.Block) {
	_, ok := txn.bufs[addr]
	if !ok {
		panic("Write: blind write")
	}
	txn.bufs[addr].dirty = true
	txn.bufs[addr].blk = blk
}

// Write of a data block.  Assumes transaction has the buf locked.
// Separate from Write() in order to support log-by-pass writes in the
// future.
func (txn *Txn) WriteData(addr uint64, blk disk.Block) {
	_, ok := txn.bufs[addr]
	if !ok {
		panic("Write: blind write")
	}
	txn.bufs[addr].dirty = true
	txn.bufs[addr].blk = blk
}

func (txn *Txn) readInodeBlock(inum uint64) disk.Block {
	if inum >= txn.fs.NInode {
		panic("readInodeBlock")
	}
	blk := txn.Read(txn.fs.inodeStart() + inum)
	return blk
}

func (txn *Txn) writeInodeBlock(inum uint64, blk disk.Block) {
	if inum >= txn.fs.NInode {
		panic("writeInodeBlock")
	}
	txn.Write(txn.fs.inodeStart()+inum, blk)
}

func (txn *Txn) releaseInodeBlock(inum uint64) {
	if inum >= txn.fs.NInode {
		panic("releaseInodeBlock")
	}
	txn.ReleaseBlock(txn.fs.inodeStart() + inum)
}

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) dirtyBufs() []*Buf {
	bufs := new([]*Buf)
	for _, buf := range txn.bufs {
		if buf.dirty {
			*bufs = append(*bufs, buf)
		}
	}
	return *bufs
}

func (txn *Txn) clearDirty(bufs []*Buf) {
	for _, b := range bufs {
		b.dirty = false
	}
}

func (txn *Txn) Pin(bufs []*Buf, n TxnNum) {
	ids := make([]uint64, len(bufs))
	for i, b := range bufs {
		ids[i] = b.blkno
	}
	txn.bc.Pin(ids, n)
}

// Commit blocks of the transaction into the log. Pin the blocks in
// the cache until installer has installed all the blocks in the log
// of this transaction.  Returns falls if trying to commit more
// buffers than fit in the log.
func (txn *Txn) CommitWait(inodes []*Inode, wait bool) bool {
	var success bool = true
	// may free an inode so must be done before Append
	txn.putInodes(inodes)

	// commit all buffers written by this transaction
	bufs := txn.dirtyBufs()
	if len(bufs) > 0 {
		n, ok := txn.log.MemAppend(bufs)
		if !ok {
			log.Printf("memappend failed\n")
			success = false
		} else {
			// must pin before waiting, otherwise unpin by
			// installer may happen before pin.  XXX
			// logger and installer run before Pin
			txn.Pin(bufs, n+1)
			if wait {
				txn.log.LogAppendWait(n)
			}
			txn.clearDirty(bufs)
		}
	}

	// release the buffers used in this transaction
	txn.release()

	// unlock all inodes used in this transaction
	unlockInodes(inodes)

	return success
}

// Append to in-memory log and wait until logger has logged this
// transaction.
// XXX commit failing
func (txn *Txn) Commit(inodes []*Inode) bool {
	return txn.CommitWait(inodes, true)
}

// XXX don't write inode if mtime is only change
func (txn *Txn) CommitData(inodes []*Inode, fh Fh) bool {
	return txn.CommitWait(inodes, true)
}

// Append to in-memory log, but don't wait for the logger to complete
// diskAppend.
func (txn *Txn) CommitUnstable(inodes []*Inode, fh Fh) bool {
	log.Printf("CommitUnstable\n")
	if len(inodes) > 1 {
		panic("CommitUnstable")
	}
	return txn.CommitWait(inodes, false)
}

// XXX Don't have to flush all data, but that is only an option if we
// do log-by-pass writes.
func (txn *Txn) CommitFh(fh Fh, inodes []*Inode) bool {
	txn.log.WaitFlushMemLog()
	txn.release()
	unlockInodes(inodes)
	return true
}

func (txn *Txn) Abort(inodes []*Inode) bool {
	log.Printf("abort\n")

	// An an abort may free an inode, which results in dirty
	// buffers that need to be written to log. So, call commit.
	return txn.Commit(inodes)
}

// Install blocks in on-disk log to their home location, and then
// unpin those blocks from cache.
// XXX would be nice to install from buffer cache, but blocks in
// buffer cache may already have been updated since previous
// transactions committed.  Maybe keep several versions
func Installer(bc *Cache, l *Log) {
	l.logLock.Lock()
	for !l.shutdown {
		blknos, txn := l.LogInstall()
		// Make space in cache by unpinning buffers that have
		// been installed
		if len(blknos) > 0 {
			log.Printf("Installed till txn %d\n", txn)
			bc.UnPin(blknos, txn)
		}
		l.condInstall.Wait()
	}
	l.logLock.Unlock()
}
