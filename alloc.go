package goose_nfs

import (
	"sync"
)

// Allocator uses a bit map to allocate and free numbers. Bit 0
// corresponds to number 1, bit 1 to 1, and so on.
type alloc struct {
	lock  *sync.RWMutex // protects next
	start uint64
	len   uint64
	next  uint64 // first number to try
	kind  kind
}

func mkAlloc(start uint64, len uint64, kind kind) *alloc {
	a := &alloc{
		lock:  new(sync.RWMutex),
		start: start,
		len:   len,
		next:  0,
		kind:  kind,
	}
	return a
}

// Free bit bn in buf
func freeBit(buf *buf, bn uint64) {
	if bn != buf.addr.off {
		panic("freeBit")
	}
	bit := bn % 8
	buf.blk[0] = buf.blk[0] & ^(1 << bit)
}

func (a *alloc) incNext(inc uint64) uint64 {
	a.lock.Lock()
	a.next = a.next + inc // inc bits
	if a.next >= a.len*NBITBLOCK {
		a.next = 0
	}
	num := a.next
	a.lock.Unlock()
	return num
}

// Returns a locked region in the bitmap with some free bits.
func (a *alloc) findFreeRegion(txn *txn) *buf {
	var buf *buf
	var num uint64
	num = a.incNext(1)
	start := num
	for {
		b := a.lockRegion(txn, num, 1)
		bit := num % 8
		dPrintf(15, "findregion: %v %d 0x%x\n", b, num, b.blk[0])
		if b.blk[0]&(1<<bit) == 0 {
			b.blk[0] = b.blk[0] | (1 << bit)
			buf = b
			break
		}
		a.unlockRegion(txn, b)
		txn.releaseBuf(b.addr)
		num = a.incNext(1)
		if num == start {
			panic("wrap around?")
		}
		continue
	}
	return buf
}

// Lock the region in the bitmap that contains n
func (a *alloc) lockRegion(txn *txn, n uint64, bits uint64) *buf {
	var buf *buf
	i := n / NBITBLOCK
	bit := n % NBITBLOCK
	addr := mkaddr(a.start+i, bit, bits)
	buf = txn.readBufLocked(addr, a.kind)
	dPrintf(15, "LockRegion: %v\n", buf)
	return buf
}

func (a *alloc) unlockRegion(txn *txn, buf *buf) {
	dPrintf(15, "UnlockRegion: %v\n", buf)
	txn.locked.del(buf.addr)
}

func (a *alloc) free(buf *buf, n uint64) {
	i := n / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}
	if buf.addr.blkno != a.start+i {
		panic("freeBlock")
	}
	freeBit(buf, n%NBITBLOCK)
}

func (a *alloc) allocNum(txn *txn) uint64 {
	var num uint64 = 0
	b := a.findFreeRegion(txn)
	if b != nil {
		b.setDirty()
		num = (b.addr.blkno-a.start)*NBITBLOCK + b.addr.off
	}
	return num
}

func (a *alloc) freeNum(txn *txn, num uint64) {
	if num == 0 {
		panic("FreeNum")
	}
	buf := a.lockRegion(txn, num, 1)
	a.free(buf, num)
	buf.setDirty()
}
