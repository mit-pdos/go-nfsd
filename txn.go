package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

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

type Txn struct {
	nfs    *Nfs
	log    *Log
	bc     *Cache   // a block cache shared between transactions
	amap   *AddrMap // the bufs that this transaction has exclusively
	fs     *FsSuper
	ic     *Cache
	balloc *Alloc
	ialloc *Alloc
	commit *Commit
	locked *AddrMap // map of bufs for on-disk objects
}

func (txn *Txn) installCache(buf *Buf, n uint64) {
	blk := buf.txn.ReadBlockCache(buf.addr.blkno)
	buf.install(blk)
	txn.bc.Pin([]uint64{buf.addr.blkno}, n)
	buf.txn.releaseBlock(buf.addr.blkno)
}

func Begin(nfs *Nfs) *Txn {
	txn := &Txn{
		nfs:    nfs,
		log:    nfs.log,
		bc:     nfs.bc,
		amap:   mkAddrMap(),
		fs:     nfs.fs,
		ic:     nfs.ic,
		balloc: nfs.balloc,
		ialloc: nfs.ialloc,
		commit: nfs.commit,
		locked: nfs.locked,
	}
	return txn
}

func (txn *Txn) readBufCore(addr Addr) *Buf {
	blk := txn.ReadBlockCache(addr.blkno)

	// make a private copy of the data in the cache
	data := make([]byte, addr.sz)
	copy(data, blk[addr.off:addr.off+addr.sz])
	buf := mkBuf(addr, data, txn)
	log.Printf("readBufCore add %v\n", buf)
	txn.amap.Add(buf)

	txn.releaseBlock(addr.blkno)
	return buf
}

func (txn *Txn) ReadBuf(addr Addr) *Buf {
	var buf *Buf
	// log.Printf("ReadBuf %v\n", addr)
	buf = txn.amap.Lookup(addr)
	if buf != nil {
		return buf
	}
	return txn.readBufCore(addr)
}

func (txn *Txn) ReadBufLocked(addr Addr) *Buf {
	var buf *Buf

	log.Printf("ReadBufLocked %v\n", addr)

	// is addr already part of this transaction?
	buf = txn.amap.Lookup(addr)
	if buf != nil {
		return buf
	}
	for {
		b := txn.readBufCore(addr)
		ok := txn.locked.LookupAdd(addr, b)
		if ok {
			buf = b
			// log.Printf("Locked %v\n", buf)
			break
		}
		continue
	}
	return buf
}

func (txn *Txn) RemBuf(buf *Buf) {
	txn.amap.Del(buf)
}

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) numberDirty() uint64 {
	return txn.amap.Dirty()
}

// Apply the update in buf to its corresponding block in the cache.
// The update in buf may only partially its block. Assume caller holds
// cache lock
func (txn *Txn) computeBlks() []*Buf {
	bufs := make([]*Buf, 0)
	for blkno, bs := range txn.amap.bufs {
		var dirty bool = false
		log.Printf("computeBlks %d %v\n", blkno, bs)
		blk := txn.ReadBlockCache(blkno)
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
			buf := mkBuf(txn.fs.Block2Addr(blkno), data, txn)
			bufs = append(bufs, buf)
			buf.Dirty()
		}
	}
	return bufs
}

// XXX Needs fixing
func (txn *Txn) releaseBufs() {
	for _, bs := range txn.amap.bufs {
		for _, b := range bs {
			log.Printf("release %v\n", b)
			if b.addr.blkno >= txn.balloc.start &&
				b.addr.blkno < txn.balloc.start+txn.balloc.len &&
				b.addr.sz == 1 {
				txn.balloc.UnlockRegion(txn, b)
			} else if b.addr.blkno >= txn.ialloc.start &&
				b.addr.blkno < txn.ialloc.start+txn.ialloc.len &&
				b.addr.sz == 1 {
				txn.ialloc.UnlockRegion(txn, b)
			} else if b.addr.sz == 4096 {
				txn.locked.Del(b)
			}
		}
	}
}

func (txn *Txn) doCommit(abort bool) (uint64, bool) {
	var n uint64 = 0
	var ok bool = false
	for !ok {
		// the following steps must be committed atomically,
		// so we hold the commit lock
		txn.commit.lock()

		bufs := txn.computeBlks()

		log.Printf("doCommit: bufs %v\n", bufs)

		// Append to the in-memory log and install+pin bufs (except
		// bitmaps) into cache
		n, ok = txn.log.MemAppend(bufs)
		if ok {
			log.Printf("install buffers")
			for _, b := range bufs {
				txn.installCache(b, n+1)
			}
		}

		txn.commit.unlock()

		if ok {
			txn.releaseBufs()
		}
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

	n, ok := txn.doCommit(abort)
	if !ok {
		log.Printf("memappend failed\n")
		success = false
	} else {
		if wait {
			txn.log.LogAppendWait(n)
		}
	}

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
