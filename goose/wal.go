package wal

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"

	"sync"
)

const LOGHDR = uint64(0)
const LOGHDR2 = uint64(1)
const LOGSTART = uint64(2)

type BlockData struct {
	Blocknum uint64
	Data     disk.Block
}

type Walog struct {
	memLock *sync.Mutex

	condLogger  *sync.Cond
	condInstall *sync.Cond

	memLog   []BlockData // in-memory log starting with memStart
	memStart uint64
	diskEnd  uint64 // next block to log to disk
	shutdown bool
}

// On-disk header in the first block of the log
type hdr struct {
	end   uint64
	addrs []uint64
}

func decodeHdr(blk disk.Block) *hdr {
	h := &hdr{
		end:   0,
		addrs: nil,
	}
	dec := NewDec(blk)
	h.end = uint64(dec.GetInt())
	h.addrs = dec.GetInts(HDRADDRS)
	return h
}

func encodeHdr(h hdr, blk disk.Block) {
	enc := NewEnc(blk)
	enc.PutInt(uint64(h.end))
	enc.PutInts(h.addrs)
}

// On-disk header in the second block of the log
type hdr2 struct {
	start uint64
}

func decodeHdr2(blk disk.Block) *hdr2 {
	h := &hdr2{
		start: 0,
	}
	dec := NewDec(blk)
	h.start = uint64(dec.GetInt())
	return h
}

func encodeHdr2(h hdr2, blk disk.Block) {
	enc := NewEnc(blk)
	enc.PutInt(uint64(h.start))
}

func (l *Walog) writeHdr(h *hdr) {
	blk := make(disk.Block, disk.BlockSize)
	encodeHdr(*h, blk)
	disk.Write(LOGHDR, blk)
}

func (l *Walog) readHdr() *hdr {
	blk := disk.Read(LOGHDR)
	h := decodeHdr(blk)
	return h
}

func (l *Walog) writeHdr2(h *hdr2) {
	blk := make(disk.Block, disk.BlockSize)
	encodeHdr2(*h, blk)
	disk.Write(LOGHDR2, blk)
}

func (l *Walog) readHdr2() *hdr2 {
	blk := disk.Read(LOGHDR2)
	h := decodeHdr2(blk)
	return h
}

//
// Installer blocks from the on-disk log to their home location.
//

func (l *Walog) installBlocks(bufs []BlockData) {
	n := uint64(len(bufs))
	for i := uint64(0); i < n; i++ {
		blkno := bufs[i].Blocknum
		blk := bufs[i].Data
		disk.Write(blkno, blk)
	}
}

// Installer holds logLock
// XXX absorp
func (l *Walog) logInstall() (uint64, uint64) {
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

func (l *Walog) installer() {
	l.memLock.Lock()
	for !l.shutdown {
		l.logInstall()
		// l.condInstall.Wait()
	}
	l.memLock.Unlock()
}

//
// Logger writes blocks from the in-memory log to the on-disk log
//

func (l *Walog) LogSz() uint64 {
	return HDRADDRS
}

func (l *Walog) logBlocks(memend uint64, memstart uint64, diskend uint64, bufs []BlockData) {
	for pos := diskend; pos < memend; pos++ {
		buf := bufs[pos-diskend]
		blk := buf.Data
		disk.Write(LOGSTART+(uint64(pos)%l.LogSz()), blk)
	}
}

// Logger holds logLock
func (l *Walog) logAppend() {
	memstart := l.memStart
	memlog := l.memLog
	memend := memstart + uint64(len(memlog))
	diskend := l.diskEnd
	newbufs := memlog[diskend-memstart:]
	if len(newbufs) == 0 {
		return
	}

	l.memLock.Unlock()

	l.logBlocks(memend, memstart, diskend, newbufs)

	addrs := make([]uint64, l.LogSz())
	for i := uint64(0); i < uint64(len(memlog)); i++ {
		pos := memstart + uint64(i)
		addrs[uint64(pos)%l.LogSz()] = memlog[i].Blocknum
	}
	newh := &hdr{
		end:   memend,
		addrs: addrs,
	}
	l.writeHdr(newh)

	l.memLock.Lock()
	l.diskEnd = memend
	// l.condLogger.Broadcast()
	// l.condInstall.Broadcast()
}

func (l *Walog) logger() {
	l.memLock.Lock()
	for !l.shutdown {
		l.logAppend()
		// l.condLogger.Wait()
	}
	l.memLock.Unlock()
}

func (l *Walog) recover() {
	h := l.readHdr()
	h2 := l.readHdr2()
	l.memStart = h2.start
	l.diskEnd = h.end
	for pos := h2.start; pos < h.end; pos++ {
		addr := h.addrs[uint64(pos)%l.LogSz()]
		blk := disk.Read(LOGSTART + (uint64(pos) % l.LogSz()))
		b := BlockData{
			Blocknum: addr,
			Data:     blk,
		}
		l.memLog = append(l.memLog, b)
	}
}

func MkLog() *Walog {
	ml := new(sync.Mutex)
	l := &Walog{
		memLock:     ml,
		condLogger:  sync.NewCond(ml),
		condInstall: sync.NewCond(ml),
		memLog:      make([]BlockData, 0),
		memStart:    0,
		diskEnd:     0,
		shutdown:    false,
	}

	l.recover()

	// TODO: do we still need to use machine.Spawn,
	//  or can we just use go statements?
	machine.Spawn(func() { l.logger() })
	machine.Spawn(func() { l.installer() })

	return l
}

func (l *Walog) memWrite(bufs []BlockData) {
	l.memLog = append(l.memLog, bufs...)
	// l.condLogger.Broadcast()
}

// Assumes caller holds memLock
// XXX absorp
func (l *Walog) doMemAppend(bufs []BlockData) uint64 {
	l.memWrite(bufs)
	txn := l.memStart + uint64(len(l.memLog))
	return txn
}

//
//  For clients of WAL
//

// Scan log for blkno. If not present, read from disk
// XXX use map
func (l *Walog) Read(blkno uint64) disk.Block {
	var blk disk.Block

	l.memLock.Lock()
	if len(l.memLog) > 0 {
		for i := len(l.memLog) - 1; ; i-- {
			buf := l.memLog[i]
			if buf.Blocknum == blkno {
				blk = make([]byte, disk.BlockSize)
				xxcopy(blk, buf.Data)
				break
			}
			if i == 0 {
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

// Append to in-memory log. Returns false, if bufs don't fit.
// Otherwise, returns the txn for this append.
func (l *Walog) MemAppend(bufs []BlockData) (uint64, bool) {
	if uint64(len(bufs)) > l.LogSz() {
		return 0, false
	}

	var txn uint64 = 0
	for {
		l.memLock.Lock()
		if uint64(l.memStart)+uint64(len(l.memLog))-uint64(l.diskEnd)+uint64(len(bufs)) > l.LogSz() {
			l.memLock.Unlock()
			// l.condLogger.Broadcast()
			// l.condInstall.Broadcast()
			continue
		}
		txn = l.doMemAppend(bufs)
		l.memLock.Unlock()
		break
	}
	return txn, true
}

// Wait until logger has appended in-memory log through txn to on-disk
// log
func (l *Walog) LogAppendWait(txn uint64) {
	l.memLock.Lock()
	for {
		if txn <= l.diskEnd {
			break
		}
		// l.condLogger.Wait()
	}
	l.memLock.Unlock()
}

// Wait until last started transaction has been appended to log.  If
// it is logged, then all preceeding transactions are also logged.
func (l *Walog) WaitFlushMemLog() {
	l.memLock.Lock()
	n := l.memStart + uint64(len(l.memLog))
	l.memLock.Unlock()

	l.LogAppendWait(n)
}

// Shutdown logger and installer
func (l *Walog) Shutdown() {
	l.memLock.Lock()
	l.shutdown = true
	// l.condLogger.Broadcast()
	// l.condInstall.Broadcast()
	l.memLock.Unlock()
}
