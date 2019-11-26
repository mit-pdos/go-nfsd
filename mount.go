package goose_nfs

import (
	"github.com/zeldovich/go-rpcgen/xdr"

	"log"
)

func (nfs *Nfs) Null(args *xdr.Void, reply *xdr.Void) error {
	log.Printf("Null\n")
	return nil
}

func (nfs *Nfs) Mount(args *Mount3, reply *Mountres3) error {
	log.Printf("Mount %v\n", args)
	reply.Fhs_status = MNT3_OK
	reply.Mountinfo.Fhandle = MkRootFh3().Data
	return nil
}

func (nfs *Nfs) Dump(args *xdr.Void, reply *Mountopt3) error {
	log.Printf("Dump\n")
	return nil
}

func (nfs *Nfs) Export(args *xdr.Void, reply *Exportsopt3) error {
	res := Exports3{}
	res.Ex_dir = "/"
	reply.P = &res
	return nil
}
