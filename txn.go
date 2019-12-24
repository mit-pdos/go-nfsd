package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

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
	bc     *cache   // a block cache shared between transactions
	amap   *addrMap // the bufs that this transaction has exclusively
	fs     *fsSuper
	balloc *alloc
	ialloc *alloc
	com    *commit
	locked *addrMap // shared map of locked addresses of disk objects
}

func begin(nfs *Nfs) *txn {
	txn := &txn{
		nfs:    nfs,
		log:    nfs.log,
		bc:     nfs.bc,
		amap:   mkaddrMap(),
		fs:     nfs.fs,
		balloc: nfs.balloc,
		ialloc: nfs.ialloc,
		com:    nfs.commit,
		locked: nfs.locked,
	}
	return txn
}

func (txn *txn) installCache(buf *buf, n txnNum) {
	blk := buf.txn.readBlockCache(buf.addr.blkno)
	buf.install(blk)
	txn.bc.pin([]uint64{buf.addr.blkno}, n)
	buf.txn.releaseBlock(buf.addr.blkno)
}

func (txn *txn) loadCache(buf *buf) {
	blk := txn.readBlockCache(buf.addr.blkno)
	byte := buf.addr.off / 8
	sz := roundUp(buf.addr.sz, 8)
	copy(buf.blk, blk[byte:byte+sz])
	dPrintf(15, "addr %v read %v %v = 0x%x\n", buf.addr, byte, sz, blk[byte:byte+1])
	txn.releaseBlock(buf.addr.blkno)
}

// ReadBufLocked acquires a lock on the address of a disk object by
// inserting the address into a shared locked map.  If it successfully
// acquires the lock, it reads a disk object from the shared block
// cache into a private buffer.  The disk objects include individual
// inodes, chunks of allocator bitmaps, and blocks.  Commit the
// release locks after installing the modified disk objects into disk
// cache.
func (txn *txn) readBufLocked(addr addr, kind kind) *buf {
	var buf *buf

	// is addr already locked for this transaction?
	buf = txn.amap.lookup(addr)
	if buf == nil {
		b := mkBufData(addr, kind, txn)
		for {
			ok := txn.locked.lookupAdd(addr, b)
			if ok {
				buf = b
				break
			}
			dPrintf(5, "%p: ReadBufLocked: try again\n", txn)
			// XXX condition variable?
			continue
		}
		txn.loadCache(buf)
		txn.amap.add(buf)

	}
	dPrintf(10, "%p: Locked %v\n", txn, buf)
	return buf
}

// Remove buffer from this transaction
func (txn *txn) releaseBuf(addr addr) {
	dPrintf(10, "%p: Unlock %v\n", txn, addr)
	txn.amap.del(addr)
}

func (txn *txn) putInodes(inodes []*inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *txn) numberDirty() uint64 {
	return txn.amap.dirty()
}

// Compute the update in buf to its corresponding block in the cache.
// The update in buf may only partially its block. Assume caller holds
// cache lock.
func (txn *txn) computeBlks() []*buf {
	var bufs = make([]*buf, 0)
	for blkno, bs := range txn.amap.bufs {
		var dirty bool = false
		dPrintf(5, "computeBlks %d %v\n", blkno, bs)
		blk := txn.readBlockCache(blkno)
		data := make([]byte, disk.BlockSize)
		copy(data, blk)
		txn.releaseBlock(blkno)
		for _, b := range bs {
			if b.install(data) {
				dirty = true
			}
		}
		if dirty {
			// construct a buf that has all changes to blkno
			buf := mkBuf(txn.fs.block2addr(blkno), 0, data, txn)
			bufs = append(bufs, buf)
			buf.setDirty()
		}
	}
	return bufs
}

func (txn *txn) unlockBuf(b *buf) {
	if b.kind == BLOCK {
		txn.locked.del(b.addr)
	} else if b.kind == INODE {
		txn.locked.del(b.addr)
	} else if b.kind == IBMAP {
		txn.ialloc.unlockRegion(txn, b)
	} else if b.kind == BBMAP {
		txn.balloc.unlockRegion(txn, b)
	}
}

func (txn *txn) releaseBufs() {
	for _, bs := range txn.amap.bufs {
		for _, b := range bs {
			dPrintf(5, "%p: unlock %v\n", txn, b)
			txn.unlockBuf(b)
		}
	}
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

		bufs := txn.computeBlks()
		if uint64(len(bufs)) > txn.log.logSz {
			txn.com.unlock()
			return 0, false
		}

		dPrintf(3, "doCommit: bufs %v\n", bufs)

		// Append to the in-memory log and install+pin bufs (except
		// bitmaps) into cache
		n, ok = txn.log.memAppend(bufs)
		if ok {
			for _, b := range bufs {
				txn.installCache(b, n+1)
			}
		}

		txn.com.unlock()

		if ok {
			txn.releaseBufs()
		}
		if !ok {
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
	txn.releaseBufs()
	return true
}

func (txn *txn) abort(inodes []*inode) bool {
	dPrintf(5, "Abort\n")

	// An an abort may free an inode, which results in dirty
	// buffers that need to be written to log. So, call commit.
	return txn.commitWait(inodes, true, true)
}

// Install blocks in on-disk log to their home location, and then
// unpin those blocks from cache.
// XXX would be nice to install from buffer cache, but blocks in
// buffer cache may already have been updated since previous
// transactions committed.  Maybe keep several versions
func installer(fs *fsSuper, bc *cache, l *walog) {
	l.logLock.Lock()
	for !l.shutdown {
		blknos, txn := l.logInstall()
		// Make space in cache by unpinning buffers that have
		// been installed, but filter out bitmap blocks.
		var bs = make([]uint64, 0)
		for _, bn := range blknos {
			if bn >= fs.inodeStart() {
				bs = append(bs, bn)
			}
		}
		if len(blknos) > 0 {
			dPrintf(5, "Installed till txn %d\n", txn)
			bc.unPin(bs, txn)
		}
		l.condInstall.Wait()
	}
	l.logLock.Unlock()
}
