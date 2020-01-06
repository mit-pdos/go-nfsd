package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

type txnNum uint64

const LOGHDR = uint64(0)
const LOGSTART = uint64(1)

type walog struct {
	// Protects in-memory-related log state
	memLock   *sync.Mutex
	logSz     uint64
	memLog    []buf  // in-memory log [memTail,memHead)
	memHead   uint64 // head of in-memory log
	memTail   uint64 // tail of in-memory log
	txnNxt    txnNum // next transaction number
	dsktxnNxt txnNum // next transaction number to install

	// Protects disk-related log state, incl. header, logtxnNxt,
	// shutdown
	logLock     *sync.Mutex
	condLogger  *sync.Cond
	condInstall *sync.Cond
	logtxnNxt   txnNum // next transaction number to log
	shutdown    bool
}

const HDRMETA = uint64(2 * 8) // space for head and tail
const HDRADDRS = (disk.BlockSize - HDRMETA) / 8
const LOGSIZE = HDRADDRS + 1 // 1 for log header

func mkLog() *walog {
	ll := new(sync.Mutex)
	l := &walog{
		memLock:     new(sync.Mutex),
		logLock:     ll,
		condLogger:  sync.NewCond(ll),
		condInstall: sync.NewCond(ll),
		logSz:       HDRADDRS,
		memLog:      make([]buf, 0),
		memHead:     0,
		memTail:     0,
		txnNxt:      0,
		logtxnNxt:   0,
		dsktxnNxt:   0,
		shutdown:    false,
	}
	dPrintf(1, "mkLog: size %d\n", l.logSz)
	l.writeHdr(0, 0, 0, l.memLog)
	return l
}

type hdr struct {
	head      uint64
	tail      uint64
	logTxnNxt txnNum // next txn to log
	addrs     []uint64
}

func decodeHdr(blk disk.Block) *hdr {
	hdr := &hdr{
		head:      0,
		tail:      0,
		logTxnNxt: 0,
		addrs:     nil,
	}
	dec := newDec(blk)
	hdr.head = dec.getInt()
	hdr.tail = dec.getInt()
	hdr.logTxnNxt = txnNum(dec.getInt())
	hdr.addrs = dec.getInts(hdr.head - hdr.tail)
	return hdr
}

func encodeHdr(hdr hdr, blk disk.Block) {
	enc := newEnc(blk)
	enc.putInt(hdr.head)
	enc.putInt(hdr.tail)
	enc.putInt(uint64(hdr.logTxnNxt))
	enc.putInts(hdr.addrs)
}

func maxLogSize() uint64 {
	return HDRADDRS * disk.BlockSize
}

func (l *walog) index(index uint64) uint64 {
	return index - l.memTail
}

func (l *walog) writeHdr(head uint64, tail uint64, dsktxnnxt txnNum, bufs []buf) {
	n := uint64(len(bufs))
	addrs := make([]uint64, n)
	if n != head-tail {
		panic("writeHdr")
	}
	for i := tail; i < head; i++ {
		addrs[l.index(i)] = bufs[l.index(i)].addr.blkno
	}
	hdr := hdr{head: head, tail: tail, logTxnNxt: dsktxnnxt, addrs: addrs}
	blk := make(disk.Block, disk.BlockSize)
	encodeHdr(hdr, blk)
	disk.Write(LOGHDR, blk)
}

func (l *walog) readHdr() *hdr {
	blk := disk.Read(LOGHDR)
	hdr := decodeHdr(blk)
	return hdr
}

func (l *walog) memWrite(bufs []*buf) {
	for _, buf := range bufs {
		l.memLog = append(l.memLog, *buf)
	}
	l.memHead = l.memHead + uint64(len(bufs))
}

// Assumes caller holds memLock
// XXX absorp
func (l *walog) doMemAppend(bufs []*buf) txnNum {
	l.memWrite(bufs)
	txn := l.txnNxt
	l.txnNxt = l.txnNxt + 1
	return txn
}

func (l *walog) readLogTxnNxt() txnNum {
	l.logLock.Lock()
	n := l.logtxnNxt
	l.logLock.Unlock()
	return n
}

func (l *walog) readtxnNxt() txnNum {
	l.memLock.Lock()
	n := l.txnNxt
	l.memLock.Unlock()
	return n
}

//
//  For clients of WAL
//

// Scan log for blkno. if not present, read from disk
func (l *walog) read(blkno uint64) disk.Block {
	var blk disk.Block

	l.memLock.Lock()
	if l.memHead > l.memTail {
		for i := l.memHead - 1; ; i-- {
			buf := l.memLog[l.index(i)]
			if buf.addr.blkno == blkno {
				blk = make([]byte, disk.BlockSize)
				copy(blk, buf.blk)
				break
			}
			if i <= l.memTail {
				break
			}
		}
	}
	l.memLock.Unlock()
	if blk == nil {
		blk = disk.Read(blkno)
	}
	return blk
}

// Append to in-memory log. Returns false, if bufs don't fit
func (l *walog) memAppend(bufs []*buf) (txnNum, bool) {
	l.memLock.Lock()
	if l.index(l.memHead)+uint64(len(bufs)) >= l.logSz {
		l.memLock.Unlock()
		return 0, false
	}
	txn := l.doMemAppend(bufs)
	l.memLock.Unlock()
	return txn, true
}

// Wait until logger has appended in-memory log through txn to on-disk
// log
func (l *walog) logAppendWait(txn txnNum) {
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
func (l *walog) waitFlushMemLog() {
	n := l.readtxnNxt() - 1
	l.logAppendWait(n)
}

func (l *walog) signalInstaller() {
	l.condInstall.Signal()
}

//
// Logger
//

func (l *walog) logBlocks(memhead uint64, diskhead uint64, bufs []buf) {
	for i := diskhead; i < memhead; i++ {
		bindex := i - diskhead
		blk := bufs[bindex].blk
		blkno := bufs[bindex].addr.blkno
		dPrintf(5, "logBlocks: %d to log block %d\n", blkno, l.index(i))
		disk.Write(LOGSTART+l.index(i), blk)
	}
}

// Logger holds logLock
func (l *walog) logAppend() {
	hdr := l.readHdr()
	l.memLock.Lock()
	memhead := l.memHead
	memtail := l.memTail
	memlog := l.memLog
	txnnxt := l.txnNxt
	if memtail != hdr.tail || memhead < hdr.head {
		panic("logAppend")
	}
	l.memLock.Unlock()

	//dPrintf("logAppend memhead %d memtail %d diskhead %d disktail %d txnnxt %d\n", memhead, memtail, hdr.head, hdr.tail, txnnxt)
	newbufs := memlog[l.index(hdr.head):l.index(memhead)]
	l.logBlocks(memhead, hdr.head, newbufs)
	l.writeHdr(memhead, memtail, txnnxt, memlog)

	l.logtxnNxt = txnnxt
}

func (l *walog) logger() {
	l.logLock.Lock()
	for !l.shutdown {
		l.logAppend()
		l.condLogger.Wait()
	}
	l.logLock.Unlock()
}

//
// Installer
//

// Install blocks in on-disk log to their home location.
func (l *walog) installer() {
	l.logLock.Lock()
	for !l.shutdown {
		blknos, txn := l.logInstall()
		if len(blknos) > 0 {
			dPrintf(5, "Installed till txn %d\n", txn)
		}
		l.condInstall.Wait()
	}
	l.logLock.Unlock()
}

func (l *walog) installBlocks(bufs []buf) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		blkno := bufs[i].addr.blkno
		blk := bufs[i].blk
		dPrintf(5, "installBlocks: write log block %d to %d\n", i, blkno)
		disk.Write(blkno, blk)
	}
}

// Installer holds logLock
// XXX absorp
func (l *walog) logInstall() ([]uint64, txnNum) {
	hdr := l.readHdr()
	bufs := l.memLog[l.index(hdr.tail):l.index(hdr.head)]
	dPrintf(1, "logInstall diskhead %d disktail %d\n", hdr.head, hdr.tail)
	l.installBlocks(bufs)
	hdr.tail = hdr.head
	l.writeHdr(hdr.head, hdr.tail, hdr.logTxnNxt, nil)
	l.memLock.Lock()

	if hdr.tail < l.memTail {
		panic("logInstall")
	}
	l.memLog = l.memLog[l.index(hdr.tail):l.index(l.memHead)]
	l.memTail = hdr.tail
	l.dsktxnNxt = hdr.logTxnNxt
	l.memLock.Unlock()
	return hdr.addrs, hdr.logTxnNxt
}

// Shutdown logger and installer
func (l *walog) doShutdown() {
	l.logLock.Lock()
	l.shutdown = true
	l.logLock.Unlock()
}
