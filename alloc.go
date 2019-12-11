package goose_nfs

import (
	"sync"
)

// Allocator uses a bit map to allocate and free numbers. Bit 0
// corresponds to number 1, bit 1 to 1, and so on.
type Alloc struct {
	lock  *sync.RWMutex // protects next
	start uint64
	len   uint64
	next  uint64 // first number to try
	kind  Kind
}

func mkAlloc(start uint64, len uint64, kind Kind) *Alloc {
	a := &Alloc{
		lock:  new(sync.RWMutex),
		start: start,
		len:   len,
		next:  0,
		kind:  kind,
	}
	return a
}

// Free bit bn in buf
func freeBit(buf *Buf, bn uint64) {
	if bn != buf.addr.off {
		panic("freeBit")
	}
	bit := bn % 8
	buf.blk[0] = buf.blk[0] & ^(1 << bit)
}

func (a *Alloc) IncNext(inc uint64) uint64 {
	a.lock.Lock()
	a.next = a.next + inc // inc bits
	if a.next >= a.len*NBITBLOCK {
		a.next = 0
	}
	num := a.next
	a.lock.Unlock()
	return num
}

func (a *Alloc) ReadNext() uint64 {
	a.lock.Lock()
	num := a.next
	a.lock.Unlock()
	return num
}

// Returns a locked region in the bitmap with some free bits.
func (a *Alloc) FindFreeRegion(txn *Txn) *Buf {
	var buf *Buf
	var num uint64
	num = a.IncNext(1)
	start := num
	for {
		b := a.LockRegion(txn, num, 1)
		bit := num % 8
		DPrintf(15, "findregion: %v %d 0x%x\n", b, num, b.blk[0])
		if b.blk[0]&(1<<bit) == 0 {
			b.blk[0] |= (1 << bit)
			buf = b
			break
		}
		a.UnlockRegion(txn, b)
		txn.ReleaseBuf(b.addr)
		num = a.IncNext(1)
		if num == start {
			panic("wrap around?")
		}
		continue
	}
	return buf
}

// Lock the region in the bitmap that contains n
func (a *Alloc) LockRegion(txn *Txn, n uint64, bits uint64) *Buf {
	var buf *Buf
	i := n / NBITBLOCK
	bit := n % NBITBLOCK
	addr := mkAddr(a.start+i, bit, bits)
	buf = txn.ReadBufLocked(addr, a.kind)
	DPrintf(15, "LockRegion: %v\n", buf)
	return buf
}

func (a *Alloc) UnlockRegion(txn *Txn, buf *Buf) {
	DPrintf(15, "UnlockRegion: %v\n", buf)
	txn.locked.Del(buf.addr)
}

func (a *Alloc) Free(buf *Buf, n uint64) {
	i := n / NBITBLOCK
	if i >= a.len {
		panic("freeBlock")
	}
	if buf.addr.blkno != a.start+i {
		panic("freeBlock")
	}
	freeBit(buf, n%NBITBLOCK)
}

func (a *Alloc) RegionAddr(n uint64, nbits uint64) Addr {
	i := n / NBITBLOCK
	bit := n % NBITBLOCK
	addr := mkAddr(a.start+i, bit, nbits)
	return addr
}

func (a *Alloc) AllocNum(txn *Txn) uint64 {
	var num uint64 = 0
	b := a.FindFreeRegion(txn)
	if b != nil {
		b.Dirty()
		num = (b.addr.blkno-a.start)*NBITBLOCK + b.addr.off
	}
	return num
}

func (a *Alloc) FreeNum(txn *Txn, num uint64) {
	if num == 0 {
		panic("FreeNum")
	}
	buf := a.LockRegion(txn, num, 1)
	a.Free(buf, num)
	buf.Dirty()
}
