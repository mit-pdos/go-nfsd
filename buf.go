package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"fmt"
)

type Kind uint64

// Type of disk objects:
const (
	BLOCK Kind = 1
	INODE Kind = 2
	IBMAP Kind = 3
	BBMAP Kind = 4
)

type Buf struct {
	kind  Kind
	addr  Addr
	blk   disk.Block
	dirty bool // has this block been written to?
	txn   *Txn
}

func mkBuf(addr Addr, kind Kind, blk disk.Block, txn *Txn) *Buf {
	b := &Buf{
		addr:  addr,
		kind:  kind,
		blk:   blk,
		dirty: false,
		txn:   txn,
	}
	return b
}

func mkBufData(addr Addr, kind Kind, txn *Txn) *Buf {
	data := make([]byte, addr.sz)
	buf := mkBuf(addr, kind, data, txn)
	return buf
}

func (buf *Buf) String() string {
	return fmt.Sprintf("%v %v %p", buf.addr, buf.dirty, buf.txn)
}

func installBits(src byte, dst byte, bit uint64, nbit uint64) byte {
	DPrintf(20, "installBits: src 0x%x dst 0x%x %d sz %d\n", src, dst, bit, nbit)
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
	DPrintf(20, "installBits -> 0x%x\n", new)
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
		nbit := Min(8-bit, n)
		srcbyte := src[0]
		dstbyte := dst[dstbyte]
		dst[dstbyte] = installBits(srcbyte, dstbyte, bit, nbit)
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
func (buf *Buf) Install(blk disk.Block) bool {
	if buf.dirty {
		copyBits(buf.blk, blk, buf.addr.off, buf.addr.sz)
	}
	return buf.dirty
}

func (buf *Buf) WriteDirect() {
	buf.Dirty()
	if buf.addr.sz == disk.BlockSize {
		disk.Write(buf.addr.blkno, buf.blk)
	} else {
		blk := disk.Read(buf.addr.blkno)
		buf.Install(blk)
		disk.Write(buf.addr.blkno, blk)
	}
}

func (buf *Buf) Dirty() {
	buf.dirty = true
}
