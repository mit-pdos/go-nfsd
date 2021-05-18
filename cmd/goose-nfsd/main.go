package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/tchajed/goose/machine/disk"
	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/xdr"

	goose_nfs "github.com/mit-pdos/goose-nfsd"
	nfstypes "github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/goose-nfsd/util/timed_disk"
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

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

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

	var d disk.Disk
	if diskfile == "" {
		d = disk.NewMemDisk(diskBlocks)
	} else {
		d, err = disk.NewFileDisk(diskfile, diskBlocks)
		if err != nil {
			panic(fmt.Errorf("could not create disk: %w", err))
		}
	}
	if dumpStats {
		d = timed_disk.New(d)
	}
	nfs := goose_nfs.MakeNfs(d)
	nfs.Unstable = unstable
	defer nfs.ShutdownNfs()

	srv := rfc1057.MakeServer()
	srv.RegisterMany(nfstypes.MOUNT_PROGRAM_MOUNT_V3_regs(nfs))
	srv.RegisterMany(nfstypes.NFS_PROGRAM_NFS_V3_regs(nfs))

	interruptSig := make(chan os.Signal, 1)
	shutdown := false
	signal.Notify(interruptSig, os.Interrupt)
	go func() {
		<-interruptSig
		shutdown = true
		listener.Close()
		if dumpStats {
			nfs.WriteOpStats(os.Stderr)
			d.(*timed_disk.Disk).WriteStats(os.Stderr)
		}
	}()

	statSig := make(chan os.Signal, 1)
	signal.Notify(statSig, syscall.SIGUSR1)
	go func() {
		for {
			<-statSig
			nfs.WriteOpStats(os.Stderr)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) && shutdown {
				util.DPrintf(1, "Shutting down server")
				break
			}
			fmt.Printf("accept: %v\n", err)
			break
		}

		go srv.Run(conn)
	}
}
