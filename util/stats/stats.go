// package stats tracks operation latencies
package stats

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/rodaine/table"
)

type Op struct {
	count uint32
	nanos uint64
}

func (op *Op) Record(start time.Time) {
	atomic.AddUint32(&op.count, 1)
	dur := time.Now().Sub(start)
	atomic.AddUint64(&op.nanos, uint64(dur.Nanoseconds()))
}

func (op Op) MicrosPerOp() float64 {
	return float64(op.count) / float64(op.nanos) / 1e3
}

func WriteTable(names []string, ops []Op, w io.Writer) {
	if len(names) != len(ops) {
		panic("mismatched names and ops lists")
	}
	tbl := table.New("op", "count", "us")
	loadedOps := make([]Op, len(ops))
	var totalOp Op
	for i := range ops {
		op := Op{
			count: atomic.LoadUint32(&ops[i].count),
			nanos: atomic.LoadUint64(&ops[i].nanos),
		}
		loadedOps[i] = op
		totalOp.count += op.count
		totalOp.nanos += op.nanos
	}
	for i, name := range names {
		micros := fmt.Sprintf("%0.1f us/op", loadedOps[i].MicrosPerOp())
		tbl.AddRow(name, loadedOps[i].count, micros)
	}
	totalMicros := float64(totalOp.nanos) / 1e3
	tbl.AddRow("total", totalOp.count, fmt.Sprintf("%0.1f us", totalMicros))
	tbl.WithWriter(w)
}

func FormatTable(names []string, ops []Op) string {
	buf := new(bytes.Buffer)
	WriteTable(names, ops, buf)
	return buf.String()
}
