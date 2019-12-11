package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

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

func findFreeBit(byteVal byte) (uint64, bool) {
	var off uint64
	var ok bool
	for bit := byte(0); bit < 8; bit++ {
		if byteVal&(byte(1)<<bit) == 0 {
			off = uint64(bit)
			ok = true
			break
		}
		continue
	}
	return off, ok
}

// Find a free bit in blk and toggle it
func findAndMark(blk disk.Block) (uint64, bool) {
	var off uint64
	var ok bool
	l := uint64(len(blk))
	for byte := uint64(0); byte < l; byte++ {
		byteVal := blk[byte]
		if byteVal == 0xff {
			continue
		}
		bit, bitOk := findFreeBit(byteVal)
		if bitOk {
			off = 8*byte + bit
			ok = true
			markBit(blk, off)
			break
		}
		// unreachable (since byte is not 0xff)
		continue
	}
	return off, ok
}

// Free bit bn in buf
func freeBit(buf *Buf, bn uint64) {
	if bn/8 != buf.addr.off {
		panic("freeBit")
	}
	bit := bn % 8
	buf.blk[0] = buf.blk[0] & ^(1 << bit)
}

// Alloc bit bn in blk
func markBit(blk disk.Block, bn uint64) {
	byte := bn / 8
	bit := bn % 8
	blk[byte] = blk[byte] | (1 << bit)
}

func (a *Alloc) IncNext() uint64 {
	a.lock.Lock()
	a.next = a.next + 8 // 1 byte at a time
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
	num = a.IncNext()
	start := num
	for {
		b := a.LockRegion(txn, num)
		if b.blk[0] != byte(0xFF) {
			buf = b
			break
		}
		a.UnlockRegion(txn, b)
		txn.ReleaseBuf(b.addr)
		num = a.IncNext()
		if num == start {
			panic("wrap around?")
			break
		}
		continue
	}
	return buf
}

// Lock the region in the bitmap that contains n
func (a *Alloc) LockRegion(txn *Txn, n uint64) *Buf {
	var buf *Buf
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	buf = txn.ReadBufLocked(addr, a.kind)
	DPrintf(10, "LockRegion: %v\n", buf)
	return buf
}

func (a *Alloc) UnlockRegion(txn *Txn, buf *Buf) {
	DPrintf(10, "UnlockRegion: %v\n", buf)
	txn.locked.Del(buf.addr)
}

func (a *Alloc) Alloc(buf *Buf) uint64 {
	var n uint64 = 0

	b, found := findAndMark(buf.blk)
	if !found {
		return n
	}
	n = (buf.addr.blkno-a.start)*NBITBLOCK + buf.addr.off*8 + b
	return n
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

func (a *Alloc) RegionAddr(n uint64) Addr {
	i := n / NBITBLOCK
	byte := (n % NBITBLOCK) / 8
	addr := mkAddr(a.start+i, byte, 1)
	return addr
}

// XXX maybe a transaction thing
func (a *Alloc) AllocMyNum(txn *Txn, blkno uint64) uint64 {
	var n uint64 = 0
	bs := txn.amap.LookupBufs(blkno)
	for _, b := range bs {
		n = a.Alloc(b)
		if n != 0 {
			break
		}
	}
	return n
}

func (a *Alloc) AllocNum(txn *Txn) uint64 {
	var n uint64 = 0
	for i := a.start; i < a.start+a.len; i++ {
		n = a.AllocMyNum(txn, i)
		if n != 0 {
			break
		}

	}
	if n == 0 {
		b := a.FindFreeRegion(txn)
		if b != nil {
			n = a.Alloc(b)
			if n == 0 {
				panic("AllocInum")
			}
			b.Dirty()
		}
	}
	return n
}

func (a *Alloc) FreeNum(txn *Txn, num uint64) {
	if num == 0 {
		panic("FreeNum")
	}
	buf := a.LockRegion(txn, num)
	a.Free(buf, num)
	buf.Dirty()
}
