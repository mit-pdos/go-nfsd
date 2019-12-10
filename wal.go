package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

type TxnNum = uint64

const LOGHDR = uint64(0)
const LOGSTART = uint64(1)

type Log struct {
	// Protects in-memory-related log state
	memLock   *sync.RWMutex
	logSz     uint64
	memLog    []Buf  // in-memory log [memTail,memHead)
	memHead   uint64 // head of in-memory log
	memTail   uint64 // tail of in-memory log
	txnNxt    TxnNum // next transaction number
	dskTxnNxt TxnNum // next transaction number to install

	// Protects disk-related log state, incl. header, logTxnNxt,
	// shutdown
	logLock     *sync.RWMutex
	condLogger  *sync.Cond
	condInstall *sync.Cond
	logTxnNxt   TxnNum // next transaction number to log
	shutdown    bool
}

const HDRMETA = uint64(2 * 8) // space for head and tail
const HDRADDRS = (disk.BlockSize - HDRMETA) / 8
const LOGSIZE = HDRADDRS + 1 // 1 for log header

func mkLog() *Log {
	ll := new(sync.RWMutex)
	l := &Log{
		memLock:     new(sync.RWMutex),
		logLock:     ll,
		condLogger:  sync.NewCond(ll),
		condInstall: sync.NewCond(ll),
		logSz:       HDRADDRS,
		memLog:      make([]Buf, 0),
		memHead:     0,
		memTail:     0,
		txnNxt:      0,
		logTxnNxt:   0,
		dskTxnNxt:   0,
		shutdown:    false,
	}
	log.Printf("mkLog: size %d\n", l.logSz)
	l.writeHdr(0, 0, 0, l.memLog)
	return l
}

type Hdr struct {
	Head      uint64
	Tail      uint64
	LogTxnNxt TxnNum // next txn to log
	Addrs     []uint64
}

func decodeHdr(blk disk.Block) Hdr {
	hdr := Hdr{}
	dec := NewDec(blk)
	hdr.Head = dec.GetInt()
	hdr.Tail = dec.GetInt()
	hdr.LogTxnNxt = dec.GetInt()
	hdr.Addrs = dec.GetInts(hdr.Head - hdr.Tail)
	return hdr
}

func encodeHdr(hdr Hdr, blk disk.Block) {
	enc := NewEnc(blk)
	enc.PutInt(hdr.Head)
	enc.PutInt(hdr.Tail)
	enc.PutInt(hdr.LogTxnNxt)
	enc.PutInts(hdr.Addrs)
}

func MaxLogSize() uint64 {
	return HDRADDRS * disk.BlockSize
}

func (l *Log) index(index uint64) uint64 {
	return index - l.memTail
}

func (l *Log) writeHdr(head uint64, tail uint64, dsktxnnxt TxnNum, bufs []Buf) {
	n := uint64(len(bufs))
	addrs := make([]uint64, n)
	if n != head-tail {
		panic("writeHdr")
	}
	for i := tail; i < head; i++ {
		addrs[l.index(i)] = bufs[l.index(i)].addr.blkno
	}
	hdr := Hdr{Head: head, Tail: tail, LogTxnNxt: dsktxnnxt, Addrs: addrs}
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
		l.memLog = append(l.memLog, *(bufs[i]))
	}
	l.memHead = l.memHead + n
}

// Assumes caller holds memLock
// XXX absorp
func (l *Log) doMemAppend(bufs []*Buf) TxnNum {
	l.memWrite(bufs)
	txn := l.txnNxt
	l.txnNxt = l.txnNxt + 1
	return txn
}

func (l *Log) readLogTxnNxt() TxnNum {
	l.logLock.Lock()
	n := l.logTxnNxt
	l.logLock.Unlock()
	return n
}

func (l *Log) readDskTxnNxt() TxnNum {
	l.memLock.Lock()
	n := l.dskTxnNxt
	l.memLock.Unlock()
	return n
}

func (l *Log) readTxnNxt() TxnNum {
	l.memLock.Lock()
	n := l.txnNxt
	l.memLock.Unlock()
	return n
}

//
//  For clients of WAL
//

// Append to in-memory log. Returns false, if bufs don't fit
func (l *Log) MemAppend(bufs []*Buf) (TxnNum, bool) {
	var txn uint64 = 0
	l.memLock.Lock()
	if l.index(l.memHead)+uint64(len(bufs)) >= l.logSz {
		l.memLock.Unlock()
		return txn, false
	}
	txn = l.doMemAppend(bufs)
	l.memLock.Unlock()
	return txn, true
}

// Wait until logger has appended in-memory log through txn to on-disk
// log
func (l *Log) LogAppendWait(txn TxnNum) {
	for {
		logtxn := l.readLogTxnNxt()
		if txn < logtxn {
			break
		}
		l.condLogger.Signal()
		continue
	}
}

// Wait until last started transaction has been appended to log.  If
// it is logged, then all preceeding transactions are also logged.
func (l *Log) WaitFlushMemLog() {
	n := l.readTxnNxt() - 1
	l.LogAppendWait(n)
}

func (l *Log) SignalInstaller() {
	l.condInstall.Signal()
}

//
// Logger
//

func (l *Log) logBlocks(memhead uint64, diskhead uint64, bufs []Buf) {
	for i := diskhead; i < memhead; i++ {
		bindex := i - diskhead
		blk := bufs[bindex].blk
		blkno := bufs[bindex].addr.blkno
		log.Printf("logBlocks: %d to log block %d\n", blkno, l.index(i))
		disk.Write(LOGSTART+l.index(i), blk)
	}
}

// Logger holds logLock
func (l *Log) logAppend() {
	hdr := l.readHdr()
	l.memLock.Lock()
	memhead := l.memHead
	memtail := l.memTail
	memlog := l.memLog
	txnnxt := l.txnNxt
	if memtail != hdr.Tail || memhead < hdr.Head {
		panic("logAppend")
	}
	l.memLock.Unlock()

	//log.Printf("logAppend memhead %d memtail %d diskhead %d disktail %d txnnxt %d\n", memhead, memtail, hdr.Head, hdr.Tail, txnnxt)
	newbufs := memlog[l.index(hdr.Head):l.index(memhead)]
	l.logBlocks(memhead, hdr.Head, newbufs)
	l.writeHdr(memhead, memtail, txnnxt, memlog)

	l.logTxnNxt = txnnxt
}

func (l *Log) Logger() {
	l.logLock.Lock()
	for !l.shutdown {
		l.logAppend()
		l.condLogger.Wait()
	}
	l.logLock.Unlock()
}

//
// For installer
//

func (l *Log) installBlocks(addrs []uint64, blks []disk.Block) {
	n := uint64(len(blks))
	for i := uint64(0); i < n; i++ {
		blkno := addrs[i]
		blk := blks[i]
		log.Printf("installBlocks: write log block %d to %d\n", i, blkno)
		disk.Write(blkno, blk)
	}
}

// Installer holds logLock
// XXX absorp
func (l *Log) LogInstall() ([]uint64, TxnNum) {
	hdr := l.readHdr()
	blks := l.readLogBlocks(hdr.Head - hdr.Tail)
	//log.Printf("logInstall diskhead %d disktail %d\n", hdr.Head, hdr.Tail)
	l.installBlocks(hdr.Addrs, blks)
	hdr.Tail = hdr.Head
	l.writeHdr(hdr.Head, hdr.Tail, hdr.LogTxnNxt, []Buf{})
	l.memLock.Lock()

	if hdr.Tail < l.memTail {
		panic("logInstall")
	}
	l.memLog = l.memLog[l.index(hdr.Tail):l.index(l.memHead)]
	l.memTail = hdr.Tail
	l.dskTxnNxt = hdr.LogTxnNxt
	l.memLock.Unlock()
	return hdr.Addrs, hdr.LogTxnNxt
}

// Shutdown logger and installer
func (l *Log) Shutdown() {
	l.logLock.Lock()
	l.shutdown = true
	l.logLock.Unlock()
}
