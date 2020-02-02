package bcache

import "github.com/tchajed/goose/machine/disk"

type Bcache struct {
	d disk.Disk
}

func MkBcache(d disk.Disk) *Bcache {
	return &Bcache{d: d}
}

func (bc *Bcache) Read(bn uint64) disk.Block {
	return bc.d.Read(bn)
}

func (bc *Bcache) Write(bn uint64, b disk.Block) {
	bc.d.Write(bn, b)
}

func (bc *Bcache) Size() uint64 {
	return bc.d.Size()
}

func (bc *Bcache) Barrier() {
	bc.d.Barrier()
}
