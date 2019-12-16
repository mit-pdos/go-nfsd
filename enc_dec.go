package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"
)

type enc struct {
	b   disk.Block
	off uint64
}

func newEnc(blk disk.Block) enc {
	return enc{b: blk, off: 0}
}

func (enc *enc) putInt32(x uint32) {
	off := enc.off
	machine.UInt32Put(enc.b[off:off+4], x)
	enc.off = enc.off + 4
}

func (enc *enc) putInt(x uint64) {
	off := enc.off
	machine.UInt64Put(enc.b[off:off+8], x)
	enc.off = enc.off + 8
}

func (enc *enc) putInts(xs []uint64) {
	// we could be slightly more efficient here by not repeatedly updating
	// the offset
	n := uint64(len(xs))
	for i := uint64(0); i < n; i++ {
		enc.putInt(xs[i])
	}
}

type dec struct {
	b   disk.Block
	off uint64
}

func newDec(b disk.Block) dec {
	return dec{b: b, off: 0}
}

func (dec *dec) getInt() uint64 {
	off := dec.off
	x := machine.UInt64Get(dec.b[off : off+8])
	dec.off = dec.off + 8
	return x
}

func (dec *dec) getInt32() uint32 {
	off := dec.off
	x := machine.UInt32Get(dec.b[off : off+4])
	dec.off = dec.off + 4
	return x
}

func (dec *dec) getInts(len uint64) []uint64 {
	xs := make([]uint64, len)
	for i := uint64(0); i < len; i++ {
		xs[i] = dec.getInt()
	}
	return xs
}

func putBytes(d []byte, b []byte) {
	for i := uint64(0); i < uint64(len(b)); i++ {
		d[i] = b[i]
	}
}

// Caller must ensure de.Name fits
func encodeDirEnt(de *dirEnt) []byte {
	d := make([]byte, DIRENTSZ)
	machine.UInt64Put(d[:8], de.inum)
	machine.UInt64Put(d[8:16], uint64(len(de.name)))
	putBytes(d[16:], []byte(de.name))
	return d
}

func decodeDirEnt(d []byte) *dirEnt {
	de := &dirEnt{}
	de.inum = machine.UInt64Get(d[:8])
	l := machine.UInt64Get(d[8:16])
	de.name = string(d[16 : 16+l])
	return de
}
