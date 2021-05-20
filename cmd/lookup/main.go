package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	goose_nfs "github.com/mit-pdos/goose-nfsd/nfs"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

const BENCHDISKSZ uint64 = 100 * 1000

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
	PLookup()
}

func Lookup(clnt *goose_nfs.NfsClient, dirfh nfstypes.Nfs_fh3, name string) {
	reply := clnt.LookupOp(dirfh, name)
	if reply.Status != nfstypes.NFS3_OK {
		panic("Lookup")
	}
	fh := reply.Resok.Object
	attr := clnt.GetattrOp(fh)
	if attr.Status != nfstypes.NFS3_OK {
		panic("Lookup")
	}
}

func PLookup() {
	const N = 1 * time.Second
	const NTHREAD = 4
	for i := 1; i <= NTHREAD; i++ {
		res := goose_nfs.Parallel(i, BENCHDISKSZ,
			func(clnt *goose_nfs.NfsClient, dirfh nfstypes.Nfs_fh3) int {
				s := strconv.Itoa(i)
				name := "x" + s
				clnt.CreateOp(dirfh, name)
				start := time.Now()
				i := 0
				for true {
					Lookup(clnt, dirfh, name)
					i++
					t := time.Now()
					elapsed := t.Sub(start)
					if elapsed >= N {
						break
					}
				}
				return i
			})
		fmt.Printf("Lookup: %d file in %d usec with %d threads\n",
			res, N.Nanoseconds()/1e3, i)

	}
}
