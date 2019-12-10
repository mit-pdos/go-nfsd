package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"fmt"
)

type Buf struct {
	slot  *Cslot
	addr  Addr
	blk   disk.Block
	blkno uint64
	dirty bool // has this block been written to?
	txn   *Txn
}

func mkBuf(addr Addr, blk disk.Block, txn *Txn) *Buf {
	b := &Buf{
		slot:  nil,
		addr:  addr,
		blk:   blk,
		blkno: addr.blkno,
		dirty: false,
		txn:   txn,
	}
	return b
}

func mkBufData(addr Addr, txn *Txn) *Buf {
	data := make([]byte, addr.sz)
	buf := mkBuf(addr, data, txn)
	return buf
}

func (buf *Buf) lock() {
	buf.slot.lock()
}

func (buf *Buf) unlock() {
	buf.slot.unlock()
}

func (buf *Buf) String() string {
	return fmt.Sprintf("%v %v %p", buf.addr, buf.dirty, buf.txn)
}

func (buf *Buf) install(blk disk.Block) bool {
	if buf.dirty {
		for i := buf.addr.off; i < buf.addr.off+buf.addr.sz; i++ {
			blk[i] = buf.blk[i-buf.addr.off]
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
