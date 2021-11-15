package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/go-journal/addr"
	"github.com/mit-pdos/go-journal/txn"
)

func testSequence(tsys *txn.Log, data []byte, tid uint64) {
	txnbuf := txn.Begin(tsys)
	for i := uint64(0); i < 16; i++ {
		txnbuf.OverWrite(addr.MkAddr(i+513, 8*tid), 8, data)
	}
	txnbuf.Commit()
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func client(tsys *txn.Log, duration time.Duration, tid uint64) int {
	data := mkdata(1)
	start := time.Now()
	i := 0
	for {
		testSequence(tsys, data, tid)
		i++
		t := time.Now()
		elapsed := t.Sub(start)
		if elapsed >= duration {
			break
		}
	}
	return i
}

func run(tsys *txn.Log, duration time.Duration, nt int) int {
	count := make(chan int)
	for i := 0; i < nt; i++ {
		go func(tid int) {
			count <- client(tsys, duration, uint64(tid))
		}(i)
	}
	n := 0
	for i := 0; i < nt; i++ {
		n += <-count
	}
	return n
}

func zeroDisk(d disk.Disk) {
	zeroblock := make([]byte, 4096)
	sz := d.Size()
	for i := uint64(0); i < sz; i++ {
		d.Write(i, zeroblock)
	}
	d.Barrier()
}

func main() {
	var err error
	var duration time.Duration
	var nthread int
	var diskfile string
	var filesizeMegabytes uint64
	flag.DurationVar(&duration, "benchtime", 10*time.Second, "time to run each iteration for")
	flag.IntVar(&nthread, "threads", 1, "number of threads to run till")
	flag.StringVar(&diskfile, "disk", "", "disk image (empty for MemDisk)")
	flag.Uint64Var(&filesizeMegabytes, "size", 400, "size of file system (in MB)")
	flag.Parse()
	if nthread < 1 {
		panic("invalid start")
	}

	diskBlocks := 1500 + filesizeMegabytes*1024/4
	var d disk.Disk
	if diskfile == "" {
		d = disk.NewMemDisk(diskBlocks)
	} else {
		d, err = disk.NewFileDisk(diskfile, diskBlocks)
		if err != nil {
			panic(fmt.Errorf("could not create disk: %w", err))
		}
	}
	zeroDisk(d)

	tsys := txn.Init(d)

	// warmup (skip if running for very little time, for example when using a
	// duration of 0s to run just one iteration)
	if duration > 500*time.Millisecond {
		run(tsys, 500*time.Millisecond, nthread)
	}

	count := run(tsys, duration, nthread)
	fmt.Printf("txn-bench: %v %v txn/sec\n", nthread, float64(count)/duration.Seconds())
}
