package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"fmt"
)

type kind uint64

// Type of disk objects:
const (
	BLOCK kind = 1
	INODE kind = 2
	IBMAP kind = 3
	BBMAP kind = 4
)

type buf struct {
	kind  kind
	addr  addr
	blk   disk.Block
	dirty bool // has this block been written to?
	txn   *txn
}

func mkBuf(addr addr, kind kind, blk disk.Block, txn *txn) *buf {
	b := &buf{
		addr:  addr,
		kind:  kind,
		blk:   blk,
		dirty: false,
		txn:   txn,
	}
	return b
}

func mkBufData(addr addr, kind kind, txn *txn) *buf {
	data := make([]byte, addr.sz)
	buf := mkBuf(addr, kind, data, txn)
	return buf
}

func (buf *buf) String() string {
	return fmt.Sprintf("%v %v %p", buf.addr, buf.dirty, buf.txn)
}

func installBits(src byte, dst byte, bit uint64, nbit uint64) byte {
	dPrintf(20, "installBits: src 0x%x dst 0x%x %d sz %d\n", src, dst, bit, nbit)
	var new byte = dst
	for i := bit; i < bit+nbit; i++ {
		if src&(1<<i) == dst&(1<<i) {
			continue
		}
		if src&(1<<i) == 0 {
			// dst is 1, but should be 0
			new = new & ^(1 << bit)
		} else {
			// dst is 0, but should be 1
			new = new | (1 << bit)
		}
	}
	dPrintf(20, "installBits -> 0x%x\n", new)
	return new
}

// copy nbits from src to dst, at dstoff in destination. dstoff is in bits.
func copyBits(src []byte, dst []byte, dstoff uint64, nbit uint64) {
	var n uint64 = nbit
	var off uint64 = 0
	var dstbyte uint64 = dstoff / 8

	// copy few last bits in first byte, if not byte aligned
	if dstoff%8 != 0 {
		bit := dstoff % 8
		nbit := min(8-bit, n)
		srcbyte := src[0]
		// TODO: which of these should be dstbyte vs dstbyte2?
		dstbyte2 := dst[dstbyte]
		dst[dstbyte2] = installBits(srcbyte, dstbyte2, bit, nbit)
		off += 8
		dstbyte += 1
		n -= nbit
	}

	// copy bytes
	sz := n / 8
	for i := off; i < off+sz; i++ {
		dst[i+dstbyte] = src[i]
	}
	n -= sz * 8
	off += sz * 8

	// copy remaining bits
	if n > 0 {
		lastbyte := off / 8
		srcbyte := src[lastbyte]
		dstbyte := dst[lastbyte+dstbyte]
		dst[lastbyte] = installBits(srcbyte, dstbyte, 0, n)
	}

}

// Install the bits from buf into blk, if buf has been modified
func (buf *buf) install(blk disk.Block) bool {
	if buf.dirty {
		copyBits(buf.blk, blk, buf.addr.off, buf.addr.sz)
	}
	return buf.dirty
}

func (buf *buf) writeDirect() {
	buf.setDirty()
	if buf.addr.sz == disk.BlockSize {
		disk.Write(buf.addr.blkno, buf.blk)
	} else {
		blk := disk.Read(buf.addr.blkno)
		buf.install(blk)
		disk.Write(buf.addr.blkno, blk)
	}
}

func (buf *buf) setDirty() {
	buf.dirty = true
}

type bufMap struct {
	addrs *addrMap
}

func mkBufMap() *bufMap {
	a := &bufMap{
		addrs: mkAddrMap(),
	}
	return a
}

func (bmap *bufMap) insert(buf *buf) {
	bmap.addrs.insert(buf.addr, buf)
}

func (bmap *bufMap) lookup(addr addr) *buf {
	e := bmap.addrs.lookup(addr)
	return e.(*buf)
}

func (bmap *bufMap) del(addr addr) {
	bmap.addrs.del(addr)
}

func (bmap *bufMap) ndirty() uint64 {
	n := 0
	bmap.addrs.apply(func(a addr, e interface{}) {
		buf := e.(*buf)
		if buf.dirty {
			n += 1
		}
	})
	return 0
}

func (bmap *bufMap) bufs() []*buf {
	bufs := make([]*buf, 0)
	bmap.addrs.apply(func(a addr, e interface{}) {
		b := e.(*buf)
		bufs = append(bufs, b)
	})
	return bufs
}
