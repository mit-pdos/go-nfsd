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

func (w *nfsWrapper) setattr(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.SETATTR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.SETATTR3res
	err = w.nfs.SetAttr(&in, &out)
	return &out, err
}

func (w *nfsWrapper) lookup(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.LOOKUP3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.LOOKUP3res
	err = w.nfs.Lookup(&in, &out)
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

func (w *nfsWrapper) read(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.READ3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.READ3res
	err = w.nfs.Read(&in, &out)
	return &out, err
}

func (w *nfsWrapper) write(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.WRITE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.WRITE3res
	err = w.nfs.Write(&in, &out)
	return &out, err
}

func (w *nfsWrapper) create(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.CREATE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.CREATE3res
	err = w.nfs.Create(&in, &out)
	return &out, err
}

func (w *nfsWrapper) mkdir(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.MKDIR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.MKDIR3res
	err = w.nfs.MakeDir(&in, &out)
	return &out, err
}

func (w *nfsWrapper) remove(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.REMOVE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.REMOVE3res
	err = w.nfs.Remove(&in, &out)
	return &out, err
}

func (w *nfsWrapper) rename(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.RENAME3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.RENAME3res
	err = w.nfs.Rename(&in, &out)
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

func (w *nfsWrapper) commit(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in goose_nfs.COMMIT3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}

	var out goose_nfs.COMMIT3res
	err = w.nfs.Commit(&in, &out)
	return &out, err
}

func registerNFS(srv *rfc1057.Server, nfs *goose_nfs.Nfs) {
	w := &nfsWrapper{nfs}

	// MOUNT

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_NULL, w.nullmount)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_MNT, w.mount)

	srv.Register(goose_nfs.MOUNT_PROGRAM, goose_nfs.MOUNT_V3,
		goose_nfs.MOUNTPROC3_EXPORT, w.export)

	// NFS

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_NULL, w.nullnfs)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_GETATTR, w.getattr)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_SETATTR, w.setattr)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_LOOKUP, w.lookup)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_ACCESS, w.access)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_READ, w.read)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_WRITE, w.write)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_CREATE, w.create)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_MKDIR, w.mkdir)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_REMOVE, w.remove)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_REMOVE, w.remove)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_RENAME, w.rename)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_FSINFO, w.fsinfo)

	srv.Register(goose_nfs.NFS_PROGRAM, goose_nfs.NFS_V3,
		goose_nfs.NFSPROC3_COMMIT, w.commit)
}
