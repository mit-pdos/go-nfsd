package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"fmt"
	"log"
	"sync"
)

type Commit struct {
	mu *sync.RWMutex
}

func mkCommit() *Commit {
	c := &Commit{
		mu: new(sync.RWMutex),
	}
	return c
}

func (c *Commit) lock() {
	c.mu.Lock()
}

func (c *Commit) unlock() {
	c.mu.Unlock()
}

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

func (buf *Buf) String() string {
	return fmt.Sprintf("%v %v", buf.blkno, buf.dirty)
}

func mkBuf(blkno uint64, blk disk.Block) *Buf {
	b := &Buf{slot: nil, blk: blk, blkno: blkno, dirty: false}
	return b
}

type Txn struct {
	log      *Log
	bc       *Cache          // a cache of Buf's shared between transactions
	bufs     map[uint64]*Buf // locked bufs in use by this transaction
	fs       *FsSuper
	ic       *Cache
	alloc    *Alloc
	commit   *Commit
	newblks  []uint64
	freeblks []uint64
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

func Begin(nfs *Nfs) *Txn {
	txn := &Txn{
		log:      nfs.log,
		bc:       nfs.bc,
		bufs:     make(map[uint64]*Buf),
		fs:       nfs.fs,
		ic:       nfs.ic,
		alloc:    nfs.alloc,
		commit:   nfs.commit,
		newblks:  make([]uint64, 0),
		freeblks: make([]uint64, 0),
	}
	return txn
}

// If Read cannot find a cache slot, wait until installer unpins
// blocks from cache: flush memlog, which may contain unstable writes,
// and signal installer.
func (txn *Txn) Read(addr uint64) disk.Block {
	if addr >= txn.fs.Size {
		panic("Read")
	}
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
			if uint64(len(txn.bufs)) >= txn.log.logSz {
				for _, b := range txn.bufs {
					log.Printf("b %d %v\n", b.blkno, b.dirty)
				}
				panic("read")
			}
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
	if addr >= txn.fs.Size {
		panic("Write")
	}
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

func (txn *Txn) AllocBlock() uint64 {
	blkno := txn.alloc.AllocBlock()
	if blkno != 0 {
		txn.newblks = append(txn.newblks, blkno)
	}
	log.Printf("alloc block %v\n", blkno)
	return blkno
}

func (txn *Txn) FreeBlock(blkno uint64) {
	if blkno == 0 {
		panic("FreeBlock")
	}
	txn.freeblks = append(txn.freeblks, blkno)
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

func (txn *Txn) doCommit(bufs []*Buf, abort bool) (uint64, bool) {
	var n uint64 = 0
	var ok bool = false
	if uint64(len(bufs)) >= txn.log.logSz {
		return 0, false
	}
	for !ok {
		// the following steps must be committed atomically,
		// so we hold the commit lock
		txn.commit.lock()

		// Compute changes to the bitmap blocks
		var bs []*Buf = bufs
		if abort {
			txn.alloc.AbortBlks(txn.newblks)
		} else {
			bitbufs := txn.alloc.CommitBmap(txn.newblks, txn.freeblks)
			log.Printf("bitbufs: %v\n", bitbufs)
			bs = append(bs, bitbufs...)
		}

		// Append to the in-memory log and pin bufs (except
		// bitmap) into cache
		n, ok = txn.log.MemAppend(bs)
		if ok {
			txn.Pin(bufs, n+1)
		}

		txn.commit.unlock()
		if !ok {
			log.Printf("doCommit: log is full; wait")
			txn.log.condLogger.Signal()
			txn.log.condInstall.Signal()
		}
	}
	return n, true

}

// Commit blocks of the transaction into the log. Pin the blocks in
// the cache until installer has installed all the blocks in the log
// of this transaction.  Returns falls if trying to commit more
// buffers than fit in the log.
func (txn *Txn) CommitWait(inodes []*Inode, wait bool, abort bool) bool {
	var success bool = true
	// may free an inode so must be done before Append
	txn.putInodes(inodes)

	// commit all buffers written by this transaction
	bufs := txn.dirtyBufs()
	if len(bufs) > 0 || len(txn.newblks) > 0 {
		n, ok := txn.doCommit(bufs, abort)
		if !ok {
			log.Printf("memappend failed\n")
			success = false
		} else {
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
func (txn *Txn) Commit(inodes []*Inode) bool {
	return txn.CommitWait(inodes, true, false)
}

// XXX don't write inode if mtime is only change
func (txn *Txn) CommitData(inodes []*Inode, fh Fh) bool {
	return txn.CommitWait(inodes, true, false)
}

// Append to in-memory log, but don't wait for the logger to complete
// diskAppend.
func (txn *Txn) CommitUnstable(inodes []*Inode, fh Fh) bool {
	log.Printf("CommitUnstable\n")
	if len(inodes) > 1 {
		panic("CommitUnstable")
	}
	return txn.CommitWait(inodes, false, false)
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
	return txn.CommitWait(inodes, true, true)
}

// Install blocks in on-disk log to their home location, and then
// unpin those blocks from cache.
// XXX would be nice to install from buffer cache, but blocks in
// buffer cache may already have been updated since previous
// transactions committed.  Maybe keep several versions
func Installer(fs *FsSuper, bc *Cache, l *Log) {
	l.logLock.Lock()
	for !l.shutdown {
		blknos, txn := l.LogInstall()
		// Make space in cache by unpinning buffers that have
		// been installed, but filter out bitmap blocks.
		bs := make([]uint64, 0)
		for _, bn := range blknos {
			if bn >= fs.inodeStart() {
				bs = append(bs, bn)
			}
		}
		if len(blknos) > 0 {
			log.Printf("Installed till txn %d\n", txn)
			bc.UnPin(bs, txn)
		}
		l.condInstall.Wait()
	}
	l.logLock.Unlock()
}
