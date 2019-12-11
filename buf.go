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

func (buf *Buf) install(blk disk.Block) bool {
	if buf.dirty {
		byte := buf.addr.off / 8
		if buf.addr.sz%8 == 0 {
			sz := buf.addr.sz / 8
			for i := byte; i < byte+sz; i++ {
				blk[i] = buf.blk[i-byte]
			}
		} else {
			bit := buf.addr.off % 8
			valbuf := buf.blk[0]
			valblk := blk[byte]
			//DPrintf(20, "valbuf 0x%x valblk 0x%x %d sz %d\n", valbuf,
			//	valblk, bit, buf.addr.sz)
			for i := bit; i < bit+buf.addr.sz; i++ {
				if valbuf&(1<<i) == valblk&(1<<i) {
					continue
				}
				if valbuf&(1<<i) == 0 {
					// valblk is 1, but should be 0
					valblk = valblk & ^(1 << bit)
				} else {
					// valblk is 0, but should be 1
					valblk = valblk | (1 << bit)
				}
			}
			blk[byte] = valblk
			DPrintf(0, "res 0x%x\n", valblk)
		}
	}
	return buf.dirty
}

func (buf *Buf) WriteDirect() {
	buf.Dirty()
	if buf.addr.sz == disk.BlockSize {
		disk.Write(buf.addr.blkno, buf.blk)
	} else {
		blk := disk.Read(buf.addr.blkno)
		buf.install(blk)
		disk.Write(buf.addr.blkno, blk)
	}
}

func (buf *Buf) Dirty() {
	buf.dirty = true
}
