package wal

import (
	"github.com/tchajed/goose/machine/disk"
)

//
// Installer blocks from the on-disk log to their home location.
//

func (l *Walog) installer() {
	l.memLock.Lock()
	for !l.shutdown {
		l.logInstall()
		// l.condInstall.Wait()
	}
	l.memLock.Unlock()
}

func (l *Walog) installBlocks(bufs []Buf) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		blkno := bufs[i].Addr.Blkno
		blk := bufs[i].Blk
		disk.Write(blkno, blk)
	}
}

// Installer holds logLock
// XXX absorp
func (l *Walog) logInstall() (uint64, LogPosition) {
	installEnd := l.diskEnd
	bufs := l.memLog[:installEnd-l.memStart]
	if len(bufs) == 0 {
		return 0, installEnd
	}

	l.memLock.Unlock()

	l.installBlocks(bufs)
	h := &hdr2{
		start: installEnd,
	}
	l.writeHdr2(h)

	l.memLock.Lock()
	if installEnd < l.memStart {
		panic("logInstall")
	}
	l.memLog = l.memLog[installEnd-l.memStart:]
	l.memStart = installEnd

	return uint64(len(bufs)), installEnd
}
