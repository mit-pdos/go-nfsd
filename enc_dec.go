package goose_nfs

import (
	"github.com/tchajed/goose/machine"
	"github.com/tchajed/goose/machine/disk"
)

type Enc struct {
	b   disk.Block
	off uint64
}

func NewEnc(blk disk.Block) Enc {
	return Enc{b: blk, off: 0}
}

func (enc *Enc) PutInt32(x uint32) {
	off := enc.off
	machine.UInt32Put(enc.b[off:off+4], x)
	enc.off = enc.off + 4
}

func (enc *Enc) PutInt(x uint64) {
	off := enc.off
	machine.UInt64Put(enc.b[off:off+8], x)
	enc.off = enc.off + 8
}

func (enc *Enc) PutInts(xs []uint64) {
	// we could be slightly more efficient here by not repeatedly updating
	// the offset
	n := uint64(len(xs))
	for i := uint64(0); i < n; i++ {
		(*enc).PutInt(xs[i])
	}
}

func (enc *Enc) PutBool(b bool) {
	off := enc.off
	if b {
		enc.b[off] = 1
	}
	if !b {
		enc.b[off] = 0
	}
	enc.off = enc.off + 1
}

func (enc *Enc) PutBytes(b []byte) {
	off := enc.off
	for i := uint64(0); i < uint64(len(b)); i++ {
		enc.b[off+i] = b[i]
	}
	enc.off = enc.off + uint64(len(b))
}

func (enc *Enc) PutString(s string) {
	(*enc).PutInt(uint64(len(s)))
	(*enc).PutBytes([]byte(s))
}

func (enc *Enc) Finish() disk.Block {
	return enc.b
}

type Dec struct {
	b   disk.Block
	off uint64
}

func NewDec(b disk.Block) Dec {
	return Dec{b: b, off: 0}
}

func (dec *Dec) GetInt() uint64 {
	off := dec.off
	x := machine.UInt64Get(dec.b[off : off+8])
	dec.off = dec.off + 8
	return x
}

func (dec *Dec) GetInt32() uint32 {
	off := dec.off
	x := machine.UInt32Get(dec.b[off : off+4])
	dec.off = dec.off + 4
	return x
}

func (dec *Dec) GetInts(len uint64) []uint64 {
	xs := make([]uint64, len)
	for i := uint64(0); i < len; i++ {
		xs[i] = (*dec).GetInt()
	}
	return xs
}

func (dec *Dec) GetBool() bool {
	off := dec.off
	x := dec.b[off]
	var b bool
	if x == 0 {
		b = false
	}
	if x == 1 {
		b = true
	}
	dec.off = dec.off + 1
	return b
}

func (dec *Dec) GetBytes(length uint64) []byte {
	off := dec.off
	bs := dec.b[off : off+length]
	dec.off = dec.off + length
	return bs
}

func (dec *Dec) GetString() string {
	length := (*dec).GetInt()
	return string((*dec).GetBytes(length))
}
