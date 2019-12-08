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

type Addr struct {
	blkno uint64
	off   uint64
	sz    uint64
}

func (a Addr) match(b Addr) bool {
	return a.blkno == b.blkno && a.off == b.off && a.sz == b.sz
}

func mkAddr(blkno uint64, off uint64, sz uint64) Addr {
	return Addr{blkno: blkno, off: off, sz: sz}
}

type Buf struct {
	slot  *Cslot
	addr  Addr
	blk   disk.Block
	blkno uint64
	dirty bool // has this block been written to?
	txn   *Txn
}

func mkBuf(addr Addr, blk disk.Block, txn *Txn) *Buf {
	b := &Buf{
		slot:  nil,
		addr:  addr,
		blk:   blk,
		blkno: addr.blkno,
		dirty: false,
		txn:   txn,
	}
	return b
}

func (buf *Buf) lock() {
	buf.slot.lock()
}

func (buf *Buf) unlock() {
	buf.slot.unlock()
}

func (buf *Buf) String() string {
	return fmt.Sprintf("%v %v", buf.addr, buf.dirty)
}

func (buf *Buf) install(blk disk.Block) bool {
	if buf.dirty {
		for i := buf.addr.off; i < buf.addr.off+buf.addr.sz; i++ {
			blk[i] = buf.blk[i-buf.addr.off]
		}
	}
	return buf.dirty
}

func (buf *Buf) WriteDirect() {
	buf.Dirty()
	if buf.addr.sz == disk.BlockSize {
		disk.Write(buf.addr.blkno, buf.blk)
	} else {
		blk := disk.Read(buf.addr.blkno)
		buf.install(blk)
		disk.Write(buf.addr.blkno, blk)
	}
}

func (buf *Buf) Dirty() {
	buf.dirty = true
}

type Txn struct {
	nfs        *Nfs
	log        *Log
	bc         *Cache            // a cache of Buf's shared between transactions
	bufs       map[uint64][]*Buf // locked bufs in use by this transaction
	fs         *FsSuper
	ic         *Cache
	balloc     *Alloc
	ialloc     *Alloc
	commit     *Commit
	newblks    []uint64
	freeblks   []uint64
	newinodes  []Inum
	freeinodes []Inum
}

// Returns a block
func loadBlock(slot *Cslot, a uint64) disk.Block {
	slot.lock()
	if slot.obj == nil {
		// blk hasn't been read yet from disk; read it and put
		// the buf with the read blk in the cache slot.
		blk := disk.Read(a)
		slot.obj = blk
	}
	blk := slot.obj.(disk.Block)
	slot.unlock()
	return blk
}

// If Read cannot find a cache slot, wait until installer unpins
// blocks from cache: flush memlog, which may contain unstable writes,
// and signal installer.
func (txn *Txn) readBlock(addr uint64) disk.Block {
	if addr >= txn.fs.Size {
		panic("Read")
	}
	var slot *Cslot
	slot = txn.bc.lookupSlot(addr)
	for slot == nil {
		log.Printf("ReadBlock: miss on %d WaitFlushMemLog and signal installer\n",
			addr)
		txn.log.WaitFlushMemLog()
		txn.log.SignalInstaller()
		if uint64(len(txn.bufs)) >= txn.log.logSz {
			log.Printf("bufs %v\n", txn.bufs)
			panic("readBlock")
		}
		// Try again; a slot should free up eventually.
		slot = txn.bc.lookupSlot(addr)
	}
	// load the block into the cache slot
	blk := loadBlock(slot, addr)
	return blk
}

func (txn *Txn) releaseBlock(blkno uint64) {
	txn.bc.freeSlot(blkno)
}

func (txn *Txn) installCache(buf *Buf, n uint64) {
	blk := buf.txn.readBlock(buf.addr.blkno)
	buf.install(blk)
	txn.bc.Pin([]uint64{buf.addr.blkno}, n)
	buf.txn.releaseBlock(buf.addr.blkno)
}

func Begin(nfs *Nfs) *Txn {
	txn := &Txn{
		nfs:        nfs,
		log:        nfs.log,
		bc:         nfs.bc,
		bufs:       make(map[uint64][]*Buf),
		fs:         nfs.fs,
		ic:         nfs.ic,
		balloc:     nfs.balloc,
		ialloc:     nfs.ialloc,
		commit:     nfs.commit,
		newblks:    make([]uint64, 0),
		freeblks:   make([]uint64, 0),
		newinodes:  make([]Inum, 0),
		freeinodes: make([]Inum, 0),
	}
	return txn
}

func (txn *Txn) ReadBuf(addr Addr) *Buf {
	var buf *Buf
	log.Printf("ReadBuf %v\n", addr)
	bs, ok := txn.bufs[addr.blkno]
	if ok {
		for _, b := range bs {
			if addr.match(b.addr) {
				buf = b
				break
			}
		}
	}
	if buf != nil {
		return buf
	}
	blk := txn.readBlock(addr.blkno)
	// make a private copy of the data in the cache
	data := make([]byte, addr.sz)
	copy(data, blk[addr.off:addr.off+addr.sz])
	buf = mkBuf(addr, data, txn)
	txn.bufs[addr.blkno] = append(txn.bufs[addr.blkno], buf)

	txn.releaseBlock(addr.blkno)

	return buf
}

// Release a not-used buffer during the transaction (e.g., during
// scanning inode or bitmap blocks that don't have free inodes or
// bits).
func (txn *Txn) ReleaseBlock(addr uint64) {
	bs, ok := txn.bufs[addr]
	if !ok {
		log.Printf("ReleaseBlock: not present")
		return
	}
	if len(bs) > 0 {
		panic("ReleaseBlock")
	}
	for _, b := range bs {
		if b.dirty {
			panic("ReleaseBlock")
		}
	}
	txn.bc.freeSlot(addr)
	delete(txn.bufs, addr)
}

func (txn *Txn) AllocBlock() uint64 {
	blkno := txn.balloc.Alloc()
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

func (txn *Txn) AllocInum() Inum {
	inum := txn.ialloc.Alloc()
	if inum != 0 {
		txn.newinodes = append(txn.newinodes, inum)
	}
	log.Printf("alloc inode %v\n", inum)
	return inum
}

func (txn *Txn) FreeInum(inum Inum) {
	if inum == 0 {
		panic("FreeInum")
	}
	txn.freeinodes = append(txn.freeinodes, inum)
}

func zeroBlock(txn *Txn, blkno uint64) {
	log.Printf("zero block %d\n", blkno)
	addr := txn.fs.Block2Addr(blkno)
	buf := txn.ReadBuf(addr)
	for i, _ := range buf.blk {
		buf.blk[i] = 0
	}
	buf.dirty = true
}

func (txn *Txn) putInodes(inodes []*Inode) {
	for _, ip := range inodes {
		ip.put(txn)
	}
}

func (txn *Txn) numberDirty() uint64 {
	var n uint64 = 0
	for _, bs := range txn.bufs {
		for _, b := range bs {
			if b.dirty {
				n += 1
			}
		}
	}
	return n
}

// Assume caller holds cache lock
func (txn *Txn) computeBlks() []*Buf {
	bufs := make([]*Buf, 0)
	for _, bs := range txn.bufs {
		var dirty bool = false
		blkno := bs[0].blkno
		blk := txn.readBlock(blkno)
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
			log.Printf("computeBlks: blk %v\n", buf)
		}
	}
	return bufs
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

		// Compute changes to the bitmap blocks
		var bs []*Buf = bufs
		if abort {
			txn.balloc.AbortNums(txn.newblks)
			txn.ialloc.AbortNums(txn.newinodes)
		} else {
			bbitbufs := txn.balloc.CommitBmap(txn.newblks, txn.freeblks)
			log.Printf("bitbufs bmap: %v\n", bbitbufs)
			bs = append(bs, bbitbufs...)
			ibitbufs := txn.ialloc.CommitBmap(txn.newinodes, txn.freeinodes)
			log.Printf("bitbufs imap: %v\n", ibitbufs)
			bs = append(bs, ibitbufs...)
		}

		// Append to the in-memory log and install+pin bufs (except
		// bitmaps) into cache
		n, ok = txn.log.MemAppend(bs)
		if ok {
			for _, b := range bufs {
				txn.installCache(b, n+1)
			}
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
