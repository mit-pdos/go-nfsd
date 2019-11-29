package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

const LOGHDR = uint64(0)
const LOGSTART = uint64(1)

type Log struct {
	logLock   *sync.RWMutex // protects on disk log
	memLock   *sync.RWMutex // protects in-memory state
	logSz     uint64
	memLog    []*Buf // in-memory log [memTail,memHead)
	memHead   uint64 // head of in-memory log
	memTail   uint64 // tail of in-memory log
	memTxnNxt uint64 // next in-memory transaction number
	logTxnNxt uint64 // next log transaction number
	shutdown  bool
}

const HDRINDICES = uint64(2 * 8)        // space for head and tail
const HDRADDRS = (512 - HDRINDICES) / 8 // XXX disk.BlockSize != 512
const LOGSIZE = HDRADDRS + 1            // 1 for HDR

func mkLog() *Log {
	l := &Log{
		logLock:   new(sync.RWMutex),
		memLock:   new(sync.RWMutex),
		logSz:     HDRADDRS,
		memLog:    make([]*Buf, 0),
		memHead:   0,
		memTail:   0,
		memTxnNxt: 0,
		logTxnNxt: 0,
		shutdown:  false,
	}
	log.Printf("mkLog: size %d\n", l.logSz)
	l.writeHdr(0, 0, l.memLog)
	return l
}

type Hdr struct {
	Head  uint64
	Tail  uint64
	Addrs []uint64
}

func decodeHdr(blk disk.Block) Hdr {
	hdr := Hdr{}
	dec := NewDec(blk)
	hdr.Head = dec.GetInt()
	hdr.Tail = dec.GetInt()
	hdr.Addrs = dec.GetInts(hdr.Head - hdr.Tail)
	return hdr
}

func encodeHdr(hdr Hdr, blk disk.Block) {
	enc := NewEnc(blk)
	enc.PutInt(hdr.Head)
	enc.PutInt(hdr.Tail)
	enc.PutInts(hdr.Addrs)
}

func (l *Log) index(index uint64) uint64 {
	return index - l.memTail
}

func (l *Log) writeHdr(head uint64, tail uint64, bufs []*Buf) {
	n := uint64(len(bufs))
	addrs := make([]uint64, n)
	if n != head-tail {
		panic("writeHdr")
	}
	for i := tail; i < head; i++ {
		addrs[l.index(i)] = bufs[l.index(i)].blkno
	}
	hdr := Hdr{Head: head, Tail: tail, Addrs: addrs}
	blk := make(disk.Block, disk.BlockSize)
	encodeHdr(hdr, blk)
	disk.Write(LOGHDR, blk)
}

func (l *Log) readHdr() Hdr {
	blk := disk.Read(LOGHDR)
	hdr := decodeHdr(blk)
	return hdr
}

func (l *Log) readLogBlocks(len uint64) []disk.Block {
	var blks = make([]disk.Block, len)
	for i := uint64(0); i < len; i++ {
		blk := disk.Read(LOGSTART + i)
		blks[i] = blk
	}
	return blks
}

func (l *Log) Read() (Hdr, []disk.Block) {
	l.logLock.Lock()
	hdr := l.readHdr()
	blks := l.readLogBlocks(hdr.Head - hdr.Tail)
	l.logLock.Unlock()
	return hdr, blks
}

func (l *Log) memWrite(bufs []*Buf) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		l.memLog = append(l.memLog, bufs[i])
	}
	l.memHead = l.memHead + n
}

func (l *Log) doMemAppend(bufs []*Buf) (bool, uint64) {
	l.memLock.Lock()
	if l.index(l.memHead)+uint64(len(bufs)) >= l.logSz {
		l.memLock.Unlock()
		return false, uint64(0)
	}
	l.memWrite(bufs)
	txn := l.memTxnNxt
	l.memTxnNxt = l.memTxnNxt + 1
	l.memLock.Unlock()
	return true, txn
}

// XXX just an atomic read?
func (l *Log) readLogTxnNxt() uint64 {
	l.memLock.Lock()
	n := l.logTxnNxt
	l.memLock.Unlock()
	return n
}

func (l *Log) diskAppendWait(txn uint64) {
	for {
		logtxn := l.readLogTxnNxt()
		if txn < logtxn {
			break
		}
		continue
	}
}

func (l *Log) MemAppend(bufs []*Buf) uint64 {
	var txn uint64 = 0
	var done bool = false
	for !done {
		done, txn = l.doMemAppend(bufs)
		if !done {
			log.Printf("out of space; wait")
		}
		continue
	}
	return txn
}

func (l *Log) Append(bufs []*Buf) bool {
	if len(bufs) == 0 {
		return true
	}
	txn := l.MemAppend(bufs)
	l.diskAppendWait(txn)
	log.Printf("Append: txn %d logged\n", txn)
	return true
}

func (l *Log) logBlocks(memhead uint64, diskhead uint64, bufs []*Buf) {
	for i := diskhead; i < memhead; i++ {
		bindex := i - diskhead
		blk := bufs[bindex].blk
		//log.Printf("logBlocks: %d to log block %v\n", bufs[bindex].blkno,
		//	l.index(i))
		disk.Write(l.index(i), blk)
	}
}

func (l *Log) logAppend() {
	l.logLock.Lock()
	hdr := l.readHdr()

	l.memLock.Lock()
	memhead := l.memHead
	memtail := l.memTail
	memlog := l.memLog
	memnxt := l.memTxnNxt
	if memtail != hdr.Tail || memhead < hdr.Head {
		panic("logAppend")
	}
	l.memLock.Unlock()

	//log.Printf("logAppend memhead %d memtail %d diskhead %d disktail %d\n",
	//	memhead, memtail, hdr.Head, hdr.Tail)
	newbufs := memlog[l.index(hdr.Head):l.index(memhead)]
	l.logBlocks(memhead, hdr.Head, newbufs)
	l.writeHdr(memhead, memtail, memlog)

	l.logTxnNxt = memnxt // XXX updating wo holding lock, atomic write?

	l.logLock.Unlock()
}

// XXX flushes too eager
func (l *Log) Logger() {
	for !l.shutdown {
		l.logAppend()
	}
}

func (l *Log) installBlocks(addrs []uint64, blks []disk.Block) {
	n := uint64(len(blks))
	for i := uint64(0); i < n; i++ {
		blkno := addrs[i]
		log.Printf("installBlocks: write log block %d to %d\n", i, blkno)
		disk.Write(blkno, blks[i])
	}
}

// XXX would be nice to install from buffer cache, but blocks in
// buffer cache may already have been updated since previous
// transactions committed.
// XXX absorp
func (l *Log) logInstall() {
	l.logLock.Lock()
	hdr := l.readHdr()
	blks := l.readLogBlocks(hdr.Head - hdr.Tail)
	log.Printf("logInstall diskhead %d disktail %d\n", hdr.Head, hdr.Tail)
	l.installBlocks(hdr.Addrs, blks)
	hdr.Tail = hdr.Head
	l.writeHdr(hdr.Head, hdr.Tail, []*Buf{})
	l.memLock.Lock()

	if hdr.Tail < l.memTail {
		panic("logInstall")
	}
	l.memLog = l.memLog[l.index(hdr.Tail):l.index(l.memHead)]
	l.memTail = hdr.Tail
	l.memLock.Unlock()
	l.logLock.Unlock()
}

// XXX installs too eager
func (l *Log) Installer() {
	for !l.shutdown {
		l.logInstall()
	}
}

func (l *Log) Shutdown() {
	// XXX protect shutdown
	l.shutdown = true
}
