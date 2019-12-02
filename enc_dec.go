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
		enc.PutInt(xs[i])
	}
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
		xs[i] = dec.GetInt()
	}
	return xs
}

func PutBytes(d []byte, b []byte) {
	for i := uint64(0); i < uint64(len(b)); i++ {
		d[i] = b[i]
	}
}

func encodeDirEnt(de *DirEnt) []byte {
	l := uint64(len(de.Name)) + 2*uint64(8)
	if l >= DIRENTSZ {
		panic("directory entry name doesn't fit")
	}
	d := make([]byte, DIRENTSZ)
	machine.UInt64Put(d[:8], de.Inum)
	machine.UInt64Put(d[8:16], uint64(len(de.Name)))
	PutBytes(d[16:], []byte(de.Name))
	return d
}

func decodeDirEnt(d []byte) *DirEnt {
	de := &DirEnt{}
	de.Inum = machine.UInt64Get(d[:8])
	l := machine.UInt64Get(d[8:16])
	de.Name = string(d[16 : 16+l])
	return de
}
