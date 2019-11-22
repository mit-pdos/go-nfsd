package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"log"
	"sync"
)

const LOGCOMMIT = uint64(0)
const LOGSTART = uint64(1)
const LOGMAXBLK = uint64(504 / 8)
const LOGEND = LOGMAXBLK + LOGSTART

type Log struct {
	logLock   *sync.RWMutex // protects on disk log
	memLock   *sync.RWMutex // protects in-memory state
	logSz     uint64
	memLog    []Buf  // in-memory log
	memLen    uint64 // length of in-memory log
	memTxnNxt uint64 // next in-memory transaction number
	logTxnNxt uint64 // next log transaction number
}

func mkLog() *Log {
	l := &Log{
		logLock:   new(sync.RWMutex),
		memLock:   new(sync.RWMutex),
		logSz:     LOGMAXBLK,
		memLog:    make([]Buf, 0),
		memLen:    0,
		memTxnNxt: 0,
		logTxnNxt: 0,
	}
	l.writeHdr(0, l.memLog)
	return l
}

type Hdr struct {
	length uint64
	addrs  []uint64 // length < LOGMAXBLOCK
}

func decodeHdr(blk disk.Block) Hdr {
	hdr := Hdr{}
	dec := NewDec(blk)
	hdr.length = dec.GetInt()
	hdr.addrs = dec.GetInts(hdr.length)
	return hdr
}

func encodeHdr(hdr Hdr) disk.Block {
	enc := NewEnc()
	enc.PutInt(hdr.length)
	enc.PutInts(hdr.addrs)
	blk := enc.Finish()
	return blk
}

func (l *Log) writeHdr(len uint64, bufs []Buf) {
	addrs := make([]uint64, len)
	for i := uint64(0); i < len; i++ {
		addrs[i] = bufs[i].blkno
	}
	hdr := Hdr{length: len, addrs: addrs}
	blk := encodeHdr(hdr)
	disk.Write(LOGCOMMIT, blk)
}

func (l *Log) readHdr() Hdr {
	blk := disk.Read(LOGCOMMIT)
	hdr := decodeHdr(blk)
	return hdr
}

func (l *Log) readBlocks(len uint64) []disk.Block {
	var blks = make([]disk.Block, 0)
	for i := uint64(0); i < len; i++ {
		blk := disk.Read(LOGSTART + i)
		blks = append(blks, blk)
	}
	return blks
}

func (l *Log) Read() []disk.Block {
	l.logLock.Lock()
	hdr := l.readHdr()
	blks := l.readBlocks(hdr.length)
	l.logLock.Unlock()
	return blks
}

func (l *Log) memWrite(bufs []Buf) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		l.memLog = append(l.memLog, bufs[i])
	}
}

func (l *Log) memAppend(bufs []Buf) (bool, uint64) {
	l.memLock.Lock()
	if l.memLen+uint64(len(bufs)) >= l.logSz {
		l.memLock.Unlock()
		return false, uint64(0)
	}
	l.memWrite(bufs)
	txn := l.memTxnNxt
	n := l.memLen + uint64(len(bufs))
	l.memLen = n
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

func (l *Log) Append(bufs []Buf) bool {
	if len(bufs) == 0 {
		return true
	}
	ok, txn := l.memAppend(bufs)
	if ok {
		l.diskAppendWait(txn)
	}
	log.Printf("txn %d logged\n", txn)
	return ok
}

func (l *Log) writeBlocks(bufs []Buf, pos uint64) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		bk := bufs[i].blk
		log.Printf("write %d to block %v\n", bufs[i].blkno, pos+i)
		disk.Write(pos+i, *bk)
	}
}

func (l *Log) diskAppend() {
	l.logLock.Lock()
	hdr := l.readHdr()

	l.memLock.Lock()
	memlen := l.memLen
	allbufs := l.memLog
	bufs := allbufs[hdr.length:]
	memnxt := l.memTxnNxt
	l.memLock.Unlock()

	// log.Printf("diskAppend mlen %d disklen %d\n", memlen, disklen)

	l.writeBlocks(bufs, hdr.length)
	l.writeHdr(memlen, allbufs)

	l.logTxnNxt = memnxt // XXX updating wo holding lock, atomic write?

	l.logLock.Unlock()
}

func (l *Log) Logger() {
	for {
		l.diskAppend()
	}
}
