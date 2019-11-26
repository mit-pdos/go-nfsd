package main

import (
	"github.com/mit-pdos/goose-nfsd"
	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/xdr"

	"log"
)

type nfsWrapper struct {
	nfs *goose_nfs.Nfs
}

func (w *nfsWrapper) null(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.Mount3

	log.Printf("wnull\n")
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.Mountres3
	err = w.nfs.Mount(&in, &out)
	return &out, err
}

func (w *nfsWrapper) mount(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	log.Printf("wmount\n")

	var in goose_nfs.Mount3
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.Mountres3
	err = w.nfs.Mount(&in, &out)
	return &out, err
}

func (w *nfsWrapper) export(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	log.Printf("wexport\n")

	var in xdr.Void
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.Exportsopt3
	err = w.nfs.Export(&in, &out)
	return &out, err
}

func (w *nfsWrapper) getattr(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.GETATTR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.GETATTR3res
	err = w.nfs.GetAttr(&in, &out)
	return &out, err
}

func registerNFS(srv *rfc1057.Server, nfs *goose_nfs.Nfs) {
	w := &nfsWrapper{nfs}

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_NULL, w.null)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_MNT, w.mount)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_EXPORT, w.export)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_GETATTR, w.getattr)
}
