package wal

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"
)

type enc struct {
	b   disk.Block
	off uint64
}

func NewEnc(blk disk.Block) enc {
	return enc{b: blk, off: 0}
}

func (enc *enc) PutInt32(x uint32) {
	off := enc.off
	machine.UInt32Put(enc.b[off:off+4], x)
	enc.off = enc.off + 4
}

func (enc *enc) PutInt(x uint64) {
	off := enc.off
	machine.UInt64Put(enc.b[off:off+8], x)
	enc.off = enc.off + 8
}

func (enc *enc) PutInts(xs []uint64) {
	// we could be slightly more efficient here by not repeatedly updating
	// the offset
	for _, x := range xs {
		enc.PutInt(x)
	}
}

type dec struct {
	b   disk.Block
	off uint64
}

func NewDec(b disk.Block) dec {
	return dec{b: b, off: 0}
}

func (dec *dec) GetInt() uint64 {
	off := dec.off
	x := machine.UInt64Get(dec.b[off : off+8])
	dec.off = dec.off + 8
	return x
}

func (dec *dec) GetInt32() uint32 {
	off := dec.off
	x := machine.UInt32Get(dec.b[off : off+4])
	dec.off = dec.off + 4
	return x
}

func (dec *dec) GetInts(len uint64) []uint64 {
	xs := make([]uint64, len)
	for i := range xs {
		xs[i] = dec.GetInt()
	}
	return xs
}

func PutBytes(d []byte, b []byte) {
	for i := uint64(0); i < uint64(len(b)); i++ {
		d[i] = b[i]
	}
}
