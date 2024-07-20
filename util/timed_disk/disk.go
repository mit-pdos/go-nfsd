package timed_disk

import (
	"io"
	"time"

	"github.com/goose-lang/goose/machine/disk"
	"github.com/mit-pdos/go-nfsd/util/stats"
)

type Disk struct {
	d   disk.Disk
	ops [3]stats.Op
}

func New(d disk.Disk) *Disk {
	return &Disk{d: d}
}

const (
	readOp int = iota
	writeOp
	barrierOp
)

var ops = []string{"disk.Read", "disk.Write", "disk.Barrier"}

// assert that Disk implements disk.Disk
var _ disk.Disk = &Disk{}

func (d *Disk) ReadTo(a uint64, b disk.Block) {
	defer d.ops[readOp].Record(time.Now())
	d.d.ReadTo(a, b)
}

func (d *Disk) Read(a uint64) disk.Block {
	buf := make(disk.Block, disk.BlockSize)
	d.ReadTo(a, buf)
	return buf
}

func (d *Disk) Write(a uint64, b disk.Block) {
	defer d.ops[writeOp].Record(time.Now())
	d.d.Write(a, b)
}

func (d *Disk) Barrier() {
	defer d.ops[barrierOp].Record(time.Now())
	d.d.Barrier()
}

func (d *Disk) Size() uint64 {
	return d.d.Size()
}

func (d *Disk) Close() {
	d.d.Close()
}

func (d *Disk) WriteStats(w io.Writer) {
	stats.WriteTable(ops, d.ops[:], w)
}

func (d *Disk) ResetStats() {
	for i := range d.ops {
		d.ops[i].Reset()
	}
}
