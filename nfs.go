package goose_nfs

import (
	"fmt"
	"path/filepath"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/util"
	"github.com/mit-pdos/goose-nfsd/barebones"

	"math/rand"
	"os"
	"strconv"
)

type Nfs struct {
	Name      *string
	Barebones *barebones.BarebonesNfs
}

func MkNfsMem(sz uint64) *Nfs {
	return MakeNfs(nil, sz)
}

func MkNfsName(name string, sz uint64) *Nfs {
	return MakeNfs(&name, sz)
}

func MkNfs(sz uint64) *Nfs {
	r := rand.Uint64()
	tmpdir := "/dev/shm"
	f, err := os.Stat(tmpdir)
	if !(err == nil && f.IsDir()) {
		tmpdir = os.TempDir()
	}
	n := filepath.Join(tmpdir, "goose"+strconv.FormatUint(r, 16)+".img")
	name := &n
	return MakeNfs(name, sz)
}

func MakeNfs(name *string, sz uint64) *Nfs {
	var d disk.Disk
	if name == nil {
		d = disk.NewMemDisk(sz)
	} else {
		util.DPrintf(1, "MakeNfs: creating file disk at %s", *name)
		var err error
		d, err = disk.NewFileDisk(*name, sz)
		if err != nil {
			panic(fmt.Errorf("could not create file disk: %v", err))
		}
	}
	bnfs := barebones.MkNfs(d)
	bnfs.Init()
	nfs := &Nfs{
		Name: name,
		Barebones: bnfs,
	}
	return nfs
}

func (nfs *Nfs) doShutdown(destroy bool) {
	util.DPrintf(1, "Shutdown %v\n", destroy)
	nfs.Barebones.Shutdown()

	if destroy && nfs.Name != nil {
		util.DPrintf(1, "Destroy %v\n", *nfs.Name)
		err := os.Remove(*nfs.Name)
		if err != nil {
			panic(err)
		}
	}

	util.DPrintf(1, "Shutdown done\n")
}

func (nfs *Nfs) ShutdownNfsDestroy() {
	nfs.doShutdown(true)
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.doShutdown(false)
}

// Terminates shrinker thread immediately
func (nfs *Nfs) Crash() {
	util.DPrintf(0, "Crash: terminate shrinker\n")
	nfs.ShutdownNfs()
}
