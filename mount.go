package goose_nfs

import (
	"log"
)

func (nfs *Nfs) Mount(args *Mount3, reply *Mountres3) error {
	log.Printf("Mount %v\n", args)
	reply.Fhs_status = MNT3_OK
	reply.Mountinfo.Fhandle = MkRootFh3().Data
	return nil
}

func (nfs *Nfs) Export(args *Mount3, reply *Exports3) error {
	reply.Ex_dir = "/"
	return nil
}
