package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"sort"
	"sync"
)

// The lock serializing transaction commit
type commit struct {
	mu *sync.Mutex
}

func mkcommit() *commit {
	c := &commit{
		mu: new(sync.Mutex),
	}
	return c
}

func (c *commit) lock() {
	c.mu.Lock()
}

func (c *commit) unlock() {
	c.mu.Unlock()
}

type txn struct {
	nfs    *Nfs
	log    *walog
	fs     *fsSuper
	balloc *alloc
	ialloc *alloc
	com    *commit
	locks  *lockMap // shared map of locks for disk objects
	bufs   *bufMap  // private map of bufs read/written by txn
}

func begin(nfs *Nfs) *txn {
	txn := &txn{
		nfs:    nfs,
		log:    nfs.log,
		fs:     nfs.fs,
		balloc: nfs.balloc,
		ialloc: nfs.ialloc,
		com:    nfs.commit,
		locks:  nfs.locks,
		bufs:   mkBufMap(),
	}
	return txn
}

func (txn *txn) load(buf *buf) {
	blk := txn.log.read(buf.addr.blkno)
	byte := buf.addr.off / 8
	sz := roundUp(buf.addr.sz, 8)
	copy(buf.blk, blk[byte:byte+sz])
	dPrintf(15, "addr %v read %v %v = 0x%x\n", buf.addr, byte, sz, blk[byte:byte+1])
}

// ReadBufLocked acquires a lock on the address of a disk object
// (i.e., inodes, chunks of allocator bitmaps, and blocks).  If it
// successfully acquires the lock, it reads a disk object from the
// shared block cache into a private buffer.  Commit releases locks
// after installing the modified disk objects into disk cache.
func (txn *txn) readBufLocked(addr addr, kind kind) *buf {

	// is addr already locked by this transaction?
	locked := txn.locks.isLocked(addr, txn)
	if !locked {
		txn.locks.acquire(addr, txn)
		buf := mkBufData(addr, kind, txn)
		txn.load(buf)
		txn.bufs.insert(buf)
	}
	buf := txn.bufs.lookup(addr)
	dPrintf(10, "%p: Locked %v\n", txn, buf)
	return buf
}

// Remove buffer from this transaction
func (txn *txn) release(addr addr) {
	dPrintf(10, "%p: Unlock %v\n", txn, addr)
	txn.locks.release(addr, txn)
	txn.bufs.del(addr)
}

func (txn *txn) putInodes(inodes []*inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *txn) numberDirty() uint64 {
	return txn.bufs.ndirty()
}

// Install the txn's bufs into their blocks.  A buf may only partially
// update a disk block. Assume caller holds commit lock.
func (txn *txn) installBufs() []*buf {
	var blks = make([]*buf, 0)

	// all bufs from this txn, sorted by blkno
	bufs := txn.bufs.bufs()
	sort.Slice(bufs, func(i, j int) bool {
		return bufs[i].addr.blkno < bufs[j].addr.blkno
	})
	l := len(bufs)
	for i := 0; i < l; {
		blkno := bufs[i].addr.blkno
		blk := txn.log.read(blkno)
		data := make([]byte, disk.BlockSize)
		copy(data, blk)
		var dirty = false
		// several bufs may contain data for different parts of the same block
		for ; i < l && blkno == bufs[i].addr.blkno; i++ {
			dPrintf(5, "computeBlks %d %v\n", blkno, bufs[i])
			if bufs[i].install(data) {
				dirty = true
			}
		}
		if dirty {
			// construct a buf that has all changes to blkno
			b := mkBuf(txn.fs.block2addr(blkno), 0, data, txn)
			blks = append(blks, b)
			b.setDirty()
		}
	}
	return blks
}

// doCommit grabs the commit log, appends to the in-memory log and
// installs changes into the cache.  Then, releases commit log, and
// locked disk objects. If it cannot commit because in-memory log is
// full, it signals the logger and installer to log and and install
// log entries, which frees up space in the in-memory log.
func (txn *txn) doCommit(abort bool) (txnNum, bool) {
	var n txnNum = 0
	var ok bool = false

	for !ok {
		// the following steps must be committed atomically,
		// so we hold the commit lock
		txn.com.lock()

		bufs := txn.installBufs()
		if uint64(len(bufs)) > txn.log.logSz {
			txn.com.unlock()
			return 0, false
		}

		dPrintf(3, "doCommit: bufs %v\n", bufs)

		n, ok = txn.log.memAppend(bufs)

		txn.com.unlock()

		if ok {
			txn.locks.releaseTxn(txn)
		} else {
			dPrintf(5, "doCommit: log is full; wait")
			txn.log.condLogger.Signal()
			txn.log.condInstall.Signal()
		}
	}
	return n, true
}

// commit blocks of the transaction into the log, and perhaps wait.
func (txn *txn) commitWait(inodes []*inode, wait bool, abort bool) bool {

	// may free an inode so must be done before commit
	txn.putInodes(inodes)

	n, ok := txn.doCommit(abort)
	if !ok {
		dPrintf(10, "memappend failed\n")
	} else {
		if wait {
			txn.log.logAppendWait(n)
		}
	}
	return ok
}

// Append to in-memory log and wait until logger has logged this
// transaction.
func (txn *txn) commit(inodes []*inode) bool {
	return txn.commitWait(inodes, true, false)
}

// XXX don't write inode if mtime is only change
func (txn *txn) commitData(inodes []*inode, fh fh) bool {
	return txn.commitWait(inodes, true, false)
}

// Append to in-memory log, but don't wait for the logger to complete
// diskAppend.
func (txn *txn) commitUnstable(inodes []*inode, fh fh) bool {
	dPrintf(5, "commitUnstable\n")
	if len(inodes) > 1 {
		panic("commitUnstable")
	}
	return txn.commitWait(inodes, false, false)
}

// XXX Don't have to flush all data, but that is only an option if we
// do log-by-pass writes.
func (txn *txn) commitFh(fh fh, inodes []*inode) bool {
	txn.log.waitFlushMemLog()
	txn.locks.releaseTxn(txn)
	return true
}

func (txn *txn) abort(inodes []*inode) bool {
	dPrintf(5, "Abort\n")

	// An an abort may free an inode, which results in dirty
	// buffers that need to be written to log. So, call commit.
	return txn.commitWait(inodes, true, true)
}
