package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/tchajed/goose/machine/disk"

	goose_nfs "github.com/mit-pdos/goose-nfsd"
	"github.com/mit-pdos/goose-nfsd/fh"
	nfstypes "github.com/mit-pdos/goose-nfsd/nfstypes"
)

const (
	FILESIZE    uint64 = 50 * 1024 * 1024
	WSIZE       uint64 = disk.BlockSize
	MB          uint64 = 1024 * 1024
	BENCHDISKSZ uint64 = 100 * 1000
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	largeFile()
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func largeFile() {
	data := mkdata(WSIZE)
	clnt := goose_nfs.MkNfsClient(BENCHDISKSZ)
	defer clnt.Shutdown()
	dir := fh.MkRootFh3()

	start := time.Now()

	name := "largefile"
	clnt.CreateOp(dir, name)
	reply := clnt.LookupOp(dir, name)
	fh := reply.Resok.Object
	n := FILESIZE / WSIZE
	for j := uint64(0); j < n; j++ {
		clnt.WriteOp(fh, j*WSIZE, data, nfstypes.UNSTABLE)
	}
	clnt.CommitOp(fh, n*WSIZE)
	attr := clnt.GetattrOp(fh)
	if uint64(attr.Resok.Obj_attributes.Size) != FILESIZE {
		panic("large")
	}

	t := time.Now()
	elapsed := t.Sub(start)
	tput := float64(FILESIZE/MB) / (float64(elapsed.Microseconds()) / 1000000)
	fmt.Printf("largefile: %v MB througput %.2f MB/s\n", FILESIZE/MB, tput)

	clnt.RemoveOp(dir, name)
}
