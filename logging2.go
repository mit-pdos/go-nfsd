package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

const LOGCOMMIT = uint64(0)
const LOGSTART = uint64(1)
const LOGMAXBLK = uint64(510)
const LOGEND = LOGMAXBLK + LOGSTART

type Log struct {
	logLock   *sync.RWMutex // protects on disk log
	memLock   *sync.RWMutex // protects in-memory state
	logSz     uint64
	memLog    []disk.Block // in-memory log
	memLen    uint64       // length of in-memory log
	memTxnNxt uint64       // next in-memory transaction number
	logTxnNxt uint64       // next log transaction number
}

func (l *Log) writeHdr(len uint64) {
	hdr := make([]byte, 4096)
	machine.UInt64Put(hdr, len)
	disk.Write(LOGCOMMIT, hdr)
}

func mkLog() *Log {
	l := &Log{
		logLock:   new(sync.RWMutex),
		memLock:   new(sync.RWMutex),
		logSz:     LOGMAXBLK,
		memLog:    make([]disk.Block, LOGMAXBLK),
		memLen:    0,
		memTxnNxt: 0,
		logTxnNxt: 0,
	}
	l.writeHdr(0)
	return l
}

func (l *Log) readHdr() uint64 {
	hdr := disk.Read(LOGCOMMIT)
	disklen := machine.UInt64Get(hdr)
	return disklen
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
	disklen := l.readHdr()
	blks := l.readBlocks(disklen)
	l.logLock.Unlock()
	return blks
}

func (l *Log) memWrite(blks []disk.Block) {
	n := uint64(len(blks))
	for i := uint64(0); i < n; i++ {
		l.memLog = append(l.memLog, blks[i])
	}
}

func (l *Log) memAppend(blks []disk.Block) (bool, uint64) {
	l.memLock.Lock()
	if l.memLen+uint64(len(blks)) >= l.logSz {
		l.memLock.Unlock()
		return false, uint64(0)
	}
	txn := l.memTxnNxt
	n := l.memLen + uint64(len(blks))
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

func (l *Log) Append(blks []disk.Block) bool {
	if len(blks) == 0 {
		return true
	}
	ok, txn := l.memAppend(blks)
	if ok {
		l.diskAppendWait(txn)
	}
	return ok
}

func (l *Log) writeBlocks(blks []disk.Block, pos uint64) {
	n := uint64(len(blks))
	for i := uint64(0); i < n; i++ {
		bk := blks[i]
		disk.Write(pos+i, bk)
	}
}

func (l *Log) diskAppend() {
	l.logLock.Lock()
	disklen := l.readHdr()

	l.memLock.Lock()
	memlen := l.memLen
	allblks := l.memLog
	blks := allblks[disklen:]
	memnxt := l.memTxnNxt
	l.memLock.Unlock()

	l.writeBlocks(blks, disklen)
	l.writeHdr(memlen)

	l.logTxnNxt = memnxt // XXX updating wo holding lock, atomic write?

	l.logLock.Unlock()
}

func (l *Log) Logger() {
	for {
		l.diskAppend()
	}
}
