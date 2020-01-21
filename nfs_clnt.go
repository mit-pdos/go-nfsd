package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

type NfsClient struct {
	srv *Nfs
}

func MkNfsClient(sz uint64) *NfsClient {
	return &NfsClient{
		srv: MkNfs(sz),
	}
}

func (clnt *NfsClient) ShutdownDestroy() {
	clnt.srv.ShutdownNfsDestroy()
}

func (clnt *NfsClient) Shutdown() {
	clnt.srv.ShutdownNfs()
}

func (clnt *NfsClient) CreateOp(fh nfstypes.Nfs_fh3, name string) nfstypes.CREATE3res {
	where := nfstypes.Diropargs3{Dir: fh, Name: nfstypes.Filename3(name)}
	how := nfstypes.Createhow3{}
	args := nfstypes.CREATE3args{Where: where, How: how}
	attr := clnt.srv.NFSPROC3_CREATE(args)
	return attr
}

func (clnt *NfsClient) LookupOp(fh nfstypes.Nfs_fh3, name string) *nfstypes.LOOKUP3res {
	what := nfstypes.Diropargs3{Dir: fh, Name: nfstypes.Filename3(name)}
	args := nfstypes.LOOKUP3args{What: what}
	reply := clnt.srv.NFSPROC3_LOOKUP(args)
	return &reply
}

func (clnt *NfsClient) GetattrOp(fh nfstypes.Nfs_fh3) *nfstypes.GETATTR3res {
	args := nfstypes.GETATTR3args{Object: fh}
	attr := clnt.srv.NFSPROC3_GETATTR(args)
	return &attr
}

func (clnt *NfsClient) WriteOp(fh nfstypes.Nfs_fh3, off uint64, data []byte, how nfstypes.Stable_how) *nfstypes.WRITE3res {
	args := nfstypes.WRITE3args{
		File:   fh,
		Offset: nfstypes.Offset3(off),
		Count:  nfstypes.Count3(len(data)),
		Stable: how,
		Data:   data}
	reply := clnt.srv.NFSPROC3_WRITE(args)
	return &reply
}

func (clnt *NfsClient) ReadOp(fh nfstypes.Nfs_fh3, off uint64, sz uint64) *nfstypes.READ3res {
	args := nfstypes.READ3args{
		File:   fh,
		Offset: nfstypes.Offset3(off),
		Count:  nfstypes.Count3(sz)}
	reply := clnt.srv.NFSPROC3_READ(args)
	return &reply
}

func (clnt *NfsClient) RemoveOp(dir nfstypes.Nfs_fh3, name string) nfstypes.REMOVE3res {
	what := nfstypes.Diropargs3{Dir: dir, Name: nfstypes.Filename3(name)}
	args := nfstypes.REMOVE3args{
		Object: what,
	}
	reply := clnt.srv.NFSPROC3_REMOVE(args)
	return reply
}

func (clnt *NfsClient) MkDirOp(dir nfstypes.Nfs_fh3, name string) nfstypes.MKDIR3res {
	where := nfstypes.Diropargs3{Dir: dir, Name: nfstypes.Filename3(name)}
	sattr := nfstypes.Sattr3{}
	args := nfstypes.MKDIR3args{Where: where, Attributes: sattr}
	attr := clnt.srv.NFSPROC3_MKDIR(args)
	return attr
}

func (clnt *NfsClient) CommitOp(fh nfstypes.Nfs_fh3, cnt uint64) *nfstypes.COMMIT3res {
	args := nfstypes.COMMIT3args{
		File:   fh,
		Offset: nfstypes.Offset3(0),
		Count:  nfstypes.Count3(cnt)}
	reply := clnt.srv.NFSPROC3_COMMIT(args)
	return &reply
}

func (clnt *NfsClient) RenameOp(fhfrom nfstypes.Nfs_fh3, from string,
	fhto nfstypes.Nfs_fh3, to string) nfstypes.Nfsstat3 {
	args := nfstypes.RENAME3args{
		From: nfstypes.Diropargs3{Dir: fhfrom, Name: nfstypes.Filename3(from)},
		To:   nfstypes.Diropargs3{Dir: fhto, Name: nfstypes.Filename3(to)},
	}
	reply := clnt.srv.NFSPROC3_RENAME(args)
	return reply.Status
}

func (clnt *NfsClient) SetattrOp(fh nfstypes.Nfs_fh3, sz uint64) nfstypes.SETATTR3res {
	size := nfstypes.Set_size3{Set_it: true, Size: nfstypes.Size3(sz)}
	attr := nfstypes.Sattr3{Size: size}
	args := nfstypes.SETATTR3args{Object: fh, New_attributes: attr}
	reply := clnt.srv.NFSPROC3_SETATTR(args)
	return reply
}

func (clnt *NfsClient) ReadDirPlusOp(dir nfstypes.Nfs_fh3, cnt uint64) nfstypes.READDIRPLUS3res {
	args := nfstypes.READDIRPLUS3args{Dir: dir, Dircount: nfstypes.Count3(100), Maxcount: nfstypes.Count3(cnt)}
	reply := clnt.srv.NFSPROC3_READDIRPLUS(args)
	return reply
}
