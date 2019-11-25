package main

import (
	"github.com/mit-pdos/goose-nfsd"
	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/xdr"
)

type nfsWrapper struct {
	nfs *goose_nfs.Nfs
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

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_GETATTR, w.getattr)
}
