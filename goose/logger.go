package wal

import (
	"github.com/tchajed/goose/machine/disk"
)

//
// Logger writes blocks from the in-memory log to the on-disk log
//

func (l *Walog) logBlocks(memend LogPosition, memstart LogPosition, diskend LogPosition, bufs []Buf) {
	for pos := diskend; pos < memend; pos++ {
		buf := bufs[pos-diskend]
		blk := buf.Blk
		disk.Write(LOGSTART+(uint64(pos)%l.LogSz()), blk)
	}
}

// Logger holds logLock
func (l *Walog) logAppend() {
	memstart := l.memStart
	memlog := l.memLog
	memend := memstart + LogPosition(len(memlog))
	diskend := l.diskEnd
	newbufs := memlog[diskend-memstart:]
	if len(newbufs) == 0 {
		return
	}

	l.memLock.Unlock()

	l.logBlocks(memend, memstart, diskend, newbufs)

	addrs := make([]uint64, l.LogSz())
	for i := uint64(0); i < uint64(len(memlog)); i++ {
		pos := memstart + LogPosition(i)
		addrs[uint64(pos)%l.LogSz()] = memlog[i].Addr.Blkno
	}
	newh := &hdr{
		end:   memend,
		addrs: addrs,
	}
	l.writeHdr(newh)

	l.memLock.Lock()
	l.diskEnd = memend
	l.condLogger.Broadcast()
	l.condInstall.Broadcast()
}

func (l *Walog) logger() {
	l.memLock.Lock()
	for !l.shutdown {
		l.logAppend()
		l.condLogger.Wait()
	}
	l.memLock.Unlock()
}
