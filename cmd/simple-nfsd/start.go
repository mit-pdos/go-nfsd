package main

import (
	"fmt"

	"github.com/goose-lang/primitive/disk"

	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/go-nfsd/simple"
)

func MakeNfs(name string) *simple.Nfs {
	sz := uint64(100 * 1024)

	var d disk.Disk
	util.DPrintf(1, "MakeNfs: creating file disk at %s", name)
	var err error
	d, err = disk.NewFileDisk(name, sz)
	if err != nil {
		panic(fmt.Errorf("could not create file disk: %v", err))
	}

	nfs := simple.MakeNfs(d)
	if nfs == nil {
		panic("could not initialize nfs")
	}

	return nfs
}
