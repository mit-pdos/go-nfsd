package bcache

import "github.com/tchajed/goose/machine/disk"

type Bcache struct{}

func MkBcache() *Bcache {
	return &Bcache{}
}

func (bc *Bcache) Read(bn uint64) disk.Block {
	return disk.Read(bn)
}

func (bc *Bcache) Write(bn uint64, b disk.Block) {
	disk.Write(bn, b)
}

func (bc *Bcache) Size() uint64 {
	return disk.Size()
}

func (bc *Bcache) Barrier() {
	// TODO: we should have this as a semantic no-op
	// disk.Barrier()
}
