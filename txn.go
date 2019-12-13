package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

// The lock serializing transaction commit
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

type Txn struct {
	nfs    *Nfs
	log    *Log
	bc     *Cache   // a block cache shared between transactions
	amap   *AddrMap // the bufs that this transaction has exclusively
	fs     *FsSuper
	balloc *Alloc
	ialloc *Alloc
	commit *Commit
	locked *AddrMap // shared map of locked addresses of disk objects
}

func Begin(nfs *Nfs) *Txn {
	txn := &Txn{
		nfs:    nfs,
		log:    nfs.log,
		bc:     nfs.bc,
		amap:   mkAddrMap(),
		fs:     nfs.fs,
		balloc: nfs.balloc,
		ialloc: nfs.ialloc,
		commit: nfs.commit,
		locked: nfs.locked,
	}
	return txn
}

func (txn *Txn) installCache(buf *Buf, n uint64) {
	blk := buf.txn.ReadBlockCache(buf.addr.blkno)
	buf.Install(blk)
	txn.bc.Pin([]uint64{buf.addr.blkno}, n)
	buf.txn.releaseBlock(buf.addr.blkno)
}

func (txn *Txn) loadCache(buf *Buf) {
	blk := txn.ReadBlockCache(buf.addr.blkno)
	byte := buf.addr.off / 8
	sz := RoundUp(buf.addr.sz, 8)
	copy(buf.blk, blk[byte:byte+sz])
	DPrintf(15, "addr %v read %v %v = 0x%x\n", buf.addr, byte, sz, blk[byte:byte+1])
	txn.releaseBlock(buf.addr.blkno)
}

// ReadBufLocked acquires a lock on the address of a disk object by
// inserting the address into a shared locked map.  If it successfully
// acquires the lock, it reads a disk object from the shared block
// cache into a private buffer.  The disk objects include individual
// inodes, chunks of allocator bitmaps, and blocks.  Commit the
// release locks after installing the modified disk objects into disk
// cache.
func (txn *Txn) ReadBufLocked(addr Addr, kind Kind) *Buf {
	var buf *Buf

	// is addr already locked for this transaction?
	buf = txn.amap.Lookup(addr)
	if buf == nil {
		b := mkBufData(addr, kind, txn)
		for {
			ok := txn.locked.LookupAdd(addr, b)
			if ok {
				buf = b
				break
			}
			DPrintf(5, "%p: ReadBufLocked: try again\n", txn)
			// XXX condition variable?
			continue
		}
		txn.loadCache(buf)
		txn.amap.Add(buf)

	}
	DPrintf(10, "%p: Locked %v\n", txn, buf)
	return buf
}

// Remove buffer from this transaction
func (txn *Txn) ReleaseBuf(addr Addr) {
	DPrintf(10, "%p: Unlock %v\n", txn, addr)
	txn.amap.Del(addr)
}

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) numberDirty() uint64 {
	return txn.amap.Dirty()
}

// Compute the update in buf to its corresponding block in the cache.
// The update in buf may only partially its block. Assume caller holds
// cache lock.
func (txn *Txn) computeBlks() []*Buf {
	bufs := make([]*Buf, 0)
	for blkno, bs := range txn.amap.bufs {
		var dirty bool = false
		DPrintf(5, "computeBlks %d %v\n", blkno, bs)
		blk := txn.ReadBlockCache(blkno)
		data := make([]byte, disk.BlockSize)
		copy(data, blk)
		txn.releaseBlock(blkno)
		for _, b := range bs {
			if b.Install(data) {
				dirty = true
			}
		}
		if dirty {
			// construct a buf that has all changes to blkno
			buf := mkBuf(txn.fs.Block2Addr(blkno), 0, data, txn)
			bufs = append(bufs, buf)
			buf.Dirty()
		}
	}
	return bufs
}

func (txn *Txn) unlockBuf(b *Buf) {
	switch b.kind {
	case BLOCK:
		txn.locked.Del(b.addr)
	case INODE:
		txn.locked.Del(b.addr)
	case IBMAP:
		txn.ialloc.UnlockRegion(txn, b)
	case BBMAP:
		txn.balloc.UnlockRegion(txn, b)
	}
}

func (txn *Txn) releaseBufs() {
	for _, bs := range txn.amap.bufs {
		for _, b := range bs {
			DPrintf(5, "%p: unlock %v\n", txn, b)
			txn.unlockBuf(b)
		}
	}
}

// doCommit grabs the commit log, appends to the in-memory log and
// installs changes into the cache.  Then, releases commit log, and
// locked disk objects. If it cannot commit because in-memory log is
// full, it signals the logger and installer to log and and install
// log entries, which frees up space in the in-memory log.
func (txn *Txn) doCommit(abort bool) (uint64, bool) {
	var n uint64 = 0
	var ok bool = false

	for !ok {
		// the following steps must be committed atomically,
		// so we hold the commit lock
		txn.commit.lock()

		bufs := txn.computeBlks()
		if uint64(len(bufs)) >= txn.log.logSz {
			break
		}

		DPrintf(3, "doCommit: bufs %v\n", bufs)

		// Append to the in-memory log and install+pin bufs (except
		// bitmaps) into cache
		n, ok = txn.log.MemAppend(bufs)
		if ok {
			for _, b := range bufs {
				txn.installCache(b, n+1)
			}
		}

		txn.commit.unlock()

		if ok {
			txn.releaseBufs()
		}
		if !ok {
			DPrintf(5, "doCommit: log is full; wait")
			txn.log.condLogger.Signal()
			txn.log.condInstall.Signal()
		}
	}
	return n, true
}

// Commit blocks of the transaction into the log, and perhaps wait.
func (txn *Txn) CommitWait(inodes []*Inode, wait bool, abort bool) bool {

	// may free an inode so must be done before commit
	txn.putInodes(inodes)

	n, ok := txn.doCommit(abort)
	if !ok {
		DPrintf(10, "memappend failed\n")
	} else {
		if wait {
			txn.log.LogAppendWait(n)
		}
	}
	return ok
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
	DPrintf(5, "CommitUnstable\n")
	if len(inodes) > 1 {
		panic("CommitUnstable")
	}
	return txn.CommitWait(inodes, false, false)
}

// XXX Don't have to flush all data, but that is only an option if we
// do log-by-pass writes.
func (txn *Txn) CommitFh(fh Fh, inodes []*Inode) bool {
	txn.log.WaitFlushMemLog()
	txn.releaseBufs()
	return true
}

func (txn *Txn) Abort(inodes []*Inode) bool {
	DPrintf(5, "Abort\n")

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
			DPrintf(5, "Installed till txn %d\n", txn)
			bc.UnPin(bs, txn)
		}
		l.condInstall.Wait()
	}
	l.logLock.Unlock()
}
