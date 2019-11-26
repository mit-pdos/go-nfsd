package main

import (
	"github.com/mit-pdos/goose-nfsd"
	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/xdr"
)

type nfsWrapper struct {
	nfs *goose_nfs.Nfs
}

func (w *nfsWrapper) nullmount(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in xdr.Void
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out xdr.Void
	err = w.nfs.NullMount(&in, &out)
	return &out, err
}

func (w *nfsWrapper) mount(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.Dirpath3
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

func (w *nfsWrapper) nullnfs(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in xdr.Void

	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out xdr.Void
	err = w.nfs.NullNFS(&in, &out)
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

func (w *nfsWrapper) access(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.ACCESS3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.ACCESS3res
	err = w.nfs.Access(&in, &out)
	return &out, err
}

func (w *nfsWrapper) fsinfo(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.FSINFO3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.FSINFO3res
	err = w.nfs.FsInfo(&in, &out)
	return &out, err
}

func registerNFS(srv *rfc1057.Server, nfs *goose_nfs.Nfs) {
	w := &nfsWrapper{nfs}

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_NULL, w.nullmount)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_MNT, w.mount)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_EXPORT, w.export)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_NULL, w.nullnfs)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_GETATTR, w.getattr)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_ACCESS, w.access)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_FSINFO, w.fsinfo)
}
