package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/xdr"

	goose_nfs "github.com/mit-pdos/goose-nfsd"
	nfstypes "github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

func pmap_set_unset(prog, vers, port uint32, setit bool) bool {
	var cred rfc1057.Opaque_auth
	cred.Flavor = rfc1057.AUTH_NONE

	pmapc, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", rfc1057.PMAP_PORT))
	if err != nil {
		panic(err)
	}
	defer pmapc.Close()
	pmap := rfc1057.MakeClient(pmapc, rfc1057.PMAP_PROG, rfc1057.PMAP_VERS)

	arg := rfc1057.Mapping{
		Prog: prog,
		Vers: vers,
		Prot: rfc1057.IPPROTO_TCP,
		Port: port,
	}

	var res xdr.Bool
	var proc uint32
	if setit {
		proc = rfc1057.PMAPPROC_SET
	} else {
		proc = rfc1057.PMAPPROC_UNSET
	}

	err = pmap.Call(proc, cred, cred, &arg, &res)
	if err != nil {
		panic(err)
	}

	return bool(res)
}

func reportStats(stats []goose_nfs.OpCount) {
	totalCount := uint32(0)
	totalNanos := uint64(0)
	for _, opCount := range stats {
		op := opCount.Op
		count := opCount.Count
		timeNanos := opCount.TimeNanos
		totalCount += count
		totalNanos += timeNanos
		microsPerOp := float64(timeNanos) / 1e3 / float64(count)
		if count > 0 {
			fmt.Fprintf(os.Stderr,
				"%14s %5d  avg: %0.1f us/op\n",
				op, count, microsPerOp)
		}
	}
	if totalCount > 0 {
		microsPerOp := float64(totalNanos) / 1e3 / float64(totalCount)
		fmt.Fprintf(os.Stderr,
			"%14s %5d  avg: %0.1f us/op\n",
			"total", totalCount, microsPerOp)
	}

}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	var unstable bool
	flag.BoolVar(&unstable, "unstable", true, "use unstable writes if requested")

	var filesizeMegabytes uint64
	flag.Uint64Var(&filesizeMegabytes, "size", 400, "size of file system (in MB)")

	var diskfile string
	flag.StringVar(&diskfile, "disk", "", "disk image (empty for MemDisk)")

	var dumpStats bool
	flag.BoolVar(&dumpStats, "stats", false, "dump stats to stderr at end")

	flag.Uint64Var(&util.Debug, "debug", 0, "debug level (higher is more verbose)")
	flag.Parse()

	diskBlocks := 1500 + filesizeMegabytes*1024/4

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	port := uint32(listener.Addr().(*net.TCPAddr).Port)

	pmap_set_unset(nfstypes.MOUNT_PROGRAM, nfstypes.MOUNT_V3, 0, false)
	ok := pmap_set_unset(nfstypes.MOUNT_PROGRAM, nfstypes.MOUNT_V3, port, true)
	if !ok {
		panic("Could not set pmap mapping for mount")
	}
	defer pmap_set_unset(nfstypes.MOUNT_PROGRAM, nfstypes.MOUNT_V3, port, false)

	pmap_set_unset(nfstypes.NFS_PROGRAM, nfstypes.NFS_V3, 0, false)
	ok = pmap_set_unset(nfstypes.NFS_PROGRAM, nfstypes.NFS_V3, port, true)
	if !ok {
		panic("Could not set pmap mapping for NFS")
	}
	defer pmap_set_unset(nfstypes.NFS_PROGRAM, nfstypes.NFS_V3, port, false)

	nfs := goose_nfs.MkNfsName(diskfile, diskBlocks)
	nfs.Unstable = unstable
	defer nfs.ShutdownNfs()

	srv := rfc1057.MakeServer()
	srv.RegisterMany(nfstypes.MOUNT_PROGRAM_MOUNT_V3_regs(nfs))
	srv.RegisterMany(nfstypes.NFS_PROGRAM_NFS_V3_regs(nfs))

	interruptSig := make(chan os.Signal, 1)
	signal.Notify(interruptSig, os.Interrupt)
	go func() {
		<-interruptSig
		listener.Close()
		if dumpStats {
			stats := nfs.GetOpStats()
			reportStats(stats)
		}
	}()

	statSig := make(chan os.Signal, 1)
	signal.Notify(statSig, syscall.SIGUSR1)
	go func() {
		for {
			<-statSig
			stats := nfs.GetOpStats()
			reportStats(stats)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("accept: %v\n", err)
			break
		}

		go srv.Run(conn)
	}
}
