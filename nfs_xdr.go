package goose_nfs

import "github.com/zeldovich/go-rpcgen/xdr"

func (v *Uint64) Xdr(xs *xdr.XdrState) {
	xdr.XdrU64(xs, (*uint64)(v))
}
func (v *Uint32) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *Filename3) Xdr(xs *xdr.XdrState) {
	xdr.XdrString(xs, int(-1), (*string)(v))
}
func (v *Nfspath3) Xdr(xs *xdr.XdrState) {
	xdr.XdrString(xs, int(-1), (*string)(v))
}
func (v *Fileid3) Xdr(xs *xdr.XdrState) {
	(*Uint64)(v).Xdr(xs)
}
func (v *Cookie3) Xdr(xs *xdr.XdrState) {
	(*Uint64)(v).Xdr(xs)
}
func (v *Cookieverf3) Xdr(xs *xdr.XdrState) {
	xdr.XdrArray(xs, (*v)[:])
}
func (v *Createverf3) Xdr(xs *xdr.XdrState) {
	xdr.XdrArray(xs, (*v)[:])
}
func (v *Writeverf3) Xdr(xs *xdr.XdrState) {
	xdr.XdrArray(xs, (*v)[:])
}
func (v *Uid3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(v).Xdr(xs)
}
func (v *Gid3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(v).Xdr(xs)
}
func (v *Size3) Xdr(xs *xdr.XdrState) {
	(*Uint64)(v).Xdr(xs)
}
func (v *Offset3) Xdr(xs *xdr.XdrState) {
	(*Uint64)(v).Xdr(xs)
}
func (v *Mode3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(v).Xdr(xs)
}
func (v *Count3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(v).Xdr(xs)
}
func (v *Nfsstat3) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *Ftype3) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *Specdata3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(&((v).Specdata1)).Xdr(xs)
	(*Uint32)(&((v).Specdata2)).Xdr(xs)
}
func (v *Nfs_fh3) Xdr(xs *xdr.XdrState) {
	xdr.XdrVarArray(xs, int(NFS3_FHSIZE), (*[]byte)(&((v).Data)))
}
func (v *Nfstime3) Xdr(xs *xdr.XdrState) {
	(*Uint32)(&((v).Seconds)).Xdr(xs)
	(*Uint32)(&((v).Nseconds)).Xdr(xs)
}
func (v *Fattr3) Xdr(xs *xdr.XdrState) {
	(*Ftype3)(&((v).Ftype)).Xdr(xs)
	(*Mode3)(&((v).Mode)).Xdr(xs)
	(*Uint32)(&((v).Nlink)).Xdr(xs)
	(*Uid3)(&((v).Uid)).Xdr(xs)
	(*Gid3)(&((v).Gid)).Xdr(xs)
	(*Size3)(&((v).Size)).Xdr(xs)
	(*Size3)(&((v).Used)).Xdr(xs)
	(*Specdata3)(&((v).Rdev)).Xdr(xs)
	(*Uint64)(&((v).Fsid)).Xdr(xs)
	(*Fileid3)(&((v).Fileid)).Xdr(xs)
	(*Nfstime3)(&((v).Atime)).Xdr(xs)
	(*Nfstime3)(&((v).Mtime)).Xdr(xs)
	(*Nfstime3)(&((v).Ctime)).Xdr(xs)
}
func (v *Post_op_attr) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Attributes_follow)))
	switch (v).Attributes_follow {
	case true:
		(*Fattr3)(&((v).Attributes)).Xdr(xs)
	case false:
	}
}
func (v *Wcc_attr) Xdr(xs *xdr.XdrState) {
	(*Size3)(&((v).Size)).Xdr(xs)
	(*Nfstime3)(&((v).Mtime)).Xdr(xs)
	(*Nfstime3)(&((v).Ctime)).Xdr(xs)
}
func (v *Pre_op_attr) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Attributes_follow)))
	switch (v).Attributes_follow {
	case true:
		(*Wcc_attr)(&((v).Attributes)).Xdr(xs)
	case false:
	}
}
func (v *Wcc_data) Xdr(xs *xdr.XdrState) {
	(*Pre_op_attr)(&((v).Before)).Xdr(xs)
	(*Post_op_attr)(&((v).After)).Xdr(xs)
}
func (v *Post_op_fh3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Handle_follows)))
	switch (v).Handle_follows {
	case true:
		(*Nfs_fh3)(&((v).Handle)).Xdr(xs)
	case false:
	}
}
func (v *Time_how) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *Set_mode3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Set_it)))
	switch (v).Set_it {
	case true:
		(*Mode3)(&((v).Mode)).Xdr(xs)
	default:
	}
}
func (v *Set_uid3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Set_it)))
	switch (v).Set_it {
	case true:
		(*Uid3)(&((v).Uid)).Xdr(xs)
	default:
	}
}
func (v *Set_gid3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Set_it)))
	switch (v).Set_it {
	case true:
		(*Gid3)(&((v).Gid)).Xdr(xs)
	default:
	}
}
func (v *Set_size3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Set_it)))
	switch (v).Set_it {
	case true:
		(*Size3)(&((v).Size)).Xdr(xs)
	default:
	}
}
func (v *Set_atime) Xdr(xs *xdr.XdrState) {
	(*Time_how)(&((v).Set_it)).Xdr(xs)
	switch (v).Set_it {
	case SET_TO_CLIENT_TIME:
		(*Nfstime3)(&((v).Atime)).Xdr(xs)
	default:
	}
}
func (v *Set_mtime) Xdr(xs *xdr.XdrState) {
	(*Time_how)(&((v).Set_it)).Xdr(xs)
	switch (v).Set_it {
	case SET_TO_CLIENT_TIME:
		(*Nfstime3)(&((v).Mtime)).Xdr(xs)
	default:
	}
}
func (v *Sattr3) Xdr(xs *xdr.XdrState) {
	(*Set_mode3)(&((v).Mode)).Xdr(xs)
	(*Set_uid3)(&((v).Uid)).Xdr(xs)
	(*Set_gid3)(&((v).Gid)).Xdr(xs)
	(*Set_size3)(&((v).Size)).Xdr(xs)
	(*Set_atime)(&((v).Atime)).Xdr(xs)
	(*Set_mtime)(&((v).Mtime)).Xdr(xs)
}
func (v *Diropargs3) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Dir)).Xdr(xs)
	(*Filename3)(&((v).Name)).Xdr(xs)
}

type NFS_PROGRAM_NFS_V3_handler interface {
	NFSPROC3_NULL()
	NFSPROC3_GETATTR(GETATTR3args) GETATTR3res
	NFSPROC3_SETATTR(SETATTR3args) SETATTR3res
	NFSPROC3_LOOKUP(LOOKUP3args) LOOKUP3res
	NFSPROC3_ACCESS(ACCESS3args) ACCESS3res
	NFSPROC3_READLINK(READLINK3args) READLINK3res
	NFSPROC3_READ(READ3args) READ3res
	NFSPROC3_WRITE(WRITE3args) WRITE3res
	NFSPROC3_CREATE(CREATE3args) CREATE3res
	NFSPROC3_MKDIR(MKDIR3args) MKDIR3res
	NFSPROC3_SYMLINK(SYMLINK3args) SYMLINK3res
	NFSPROC3_MKNOD(MKNOD3args) MKNOD3res
	NFSPROC3_REMOVE(REMOVE3args) REMOVE3res
	NFSPROC3_RMDIR(RMDIR3args) RMDIR3res
	NFSPROC3_RENAME(RENAME3args) RENAME3res
	NFSPROC3_LINK(LINK3args) LINK3res
	NFSPROC3_READDIR(READDIR3args) READDIR3res
	NFSPROC3_READDIRPLUS(READDIRPLUS3args) READDIRPLUS3res
	NFSPROC3_FSSTAT(FSSTAT3args) FSSTAT3res
	NFSPROC3_FSINFO(FSINFO3args) FSINFO3res
	NFSPROC3_PATHCONF(PATHCONF3args) PATHCONF3res
	NFSPROC3_COMMIT(COMMIT3args) COMMIT3res
}
type NFS_PROGRAM_NFS_V3_handler_wrapper struct {
	h NFS_PROGRAM_NFS_V3_handler
}

func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_NULL(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var out xdr.Void
	w.h.NFSPROC3_NULL()
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_GETATTR(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in GETATTR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out GETATTR3res
	out = w.h.NFSPROC3_GETATTR(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_SETATTR(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in SETATTR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out SETATTR3res
	out = w.h.NFSPROC3_SETATTR(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_LOOKUP(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in LOOKUP3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out LOOKUP3res
	out = w.h.NFSPROC3_LOOKUP(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_ACCESS(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in ACCESS3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out ACCESS3res
	out = w.h.NFSPROC3_ACCESS(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_READLINK(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in READLINK3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out READLINK3res
	out = w.h.NFSPROC3_READLINK(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_READ(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in READ3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out READ3res
	out = w.h.NFSPROC3_READ(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_WRITE(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in WRITE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out WRITE3res
	out = w.h.NFSPROC3_WRITE(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_CREATE(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in CREATE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out CREATE3res
	out = w.h.NFSPROC3_CREATE(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_MKDIR(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in MKDIR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out MKDIR3res
	out = w.h.NFSPROC3_MKDIR(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_SYMLINK(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in SYMLINK3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out SYMLINK3res
	out = w.h.NFSPROC3_SYMLINK(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_MKNOD(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in MKNOD3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out MKNOD3res
	out = w.h.NFSPROC3_MKNOD(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_REMOVE(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in REMOVE3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out REMOVE3res
	out = w.h.NFSPROC3_REMOVE(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_RMDIR(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in RMDIR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out RMDIR3res
	out = w.h.NFSPROC3_RMDIR(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_RENAME(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in RENAME3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out RENAME3res
	out = w.h.NFSPROC3_RENAME(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_LINK(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in LINK3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out LINK3res
	out = w.h.NFSPROC3_LINK(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_READDIR(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in READDIR3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out READDIR3res
	out = w.h.NFSPROC3_READDIR(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_READDIRPLUS(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in READDIRPLUS3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out READDIRPLUS3res
	out = w.h.NFSPROC3_READDIRPLUS(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_FSSTAT(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in FSSTAT3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out FSSTAT3res
	out = w.h.NFSPROC3_FSSTAT(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_FSINFO(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in FSINFO3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out FSINFO3res
	out = w.h.NFSPROC3_FSINFO(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_PATHCONF(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in PATHCONF3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out PATHCONF3res
	out = w.h.NFSPROC3_PATHCONF(in)
	return &out, nil
}
func (w *NFS_PROGRAM_NFS_V3_handler_wrapper) NFSPROC3_COMMIT(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in COMMIT3args
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out COMMIT3res
	out = w.h.NFSPROC3_COMMIT(in)
	return &out, nil
}
func NFS_PROGRAM_NFS_V3_regs(h NFS_PROGRAM_NFS_V3_handler) []xdr.ProcRegistration {
	w := &NFS_PROGRAM_NFS_V3_handler_wrapper{h}
	return []xdr.ProcRegistration{
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_NULL,
			Handler: w.NFSPROC3_NULL,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_GETATTR,
			Handler: w.NFSPROC3_GETATTR,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_SETATTR,
			Handler: w.NFSPROC3_SETATTR,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_LOOKUP,
			Handler: w.NFSPROC3_LOOKUP,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_ACCESS,
			Handler: w.NFSPROC3_ACCESS,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_READLINK,
			Handler: w.NFSPROC3_READLINK,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_READ,
			Handler: w.NFSPROC3_READ,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_WRITE,
			Handler: w.NFSPROC3_WRITE,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_CREATE,
			Handler: w.NFSPROC3_CREATE,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_MKDIR,
			Handler: w.NFSPROC3_MKDIR,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_SYMLINK,
			Handler: w.NFSPROC3_SYMLINK,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_MKNOD,
			Handler: w.NFSPROC3_MKNOD,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_REMOVE,
			Handler: w.NFSPROC3_REMOVE,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_RMDIR,
			Handler: w.NFSPROC3_RMDIR,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_RENAME,
			Handler: w.NFSPROC3_RENAME,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_LINK,
			Handler: w.NFSPROC3_LINK,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_READDIR,
			Handler: w.NFSPROC3_READDIR,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_READDIRPLUS,
			Handler: w.NFSPROC3_READDIRPLUS,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_FSSTAT,
			Handler: w.NFSPROC3_FSSTAT,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_FSINFO,
			Handler: w.NFSPROC3_FSINFO,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_PATHCONF,
			Handler: w.NFSPROC3_PATHCONF,
		},
		xdr.ProcRegistration{
			Prog:    NFS_PROGRAM,
			Vers:    NFS_V3,
			Proc:    NFSPROC3_COMMIT,
			Handler: w.NFSPROC3_COMMIT,
		},
	}
}
func (v *GETATTR3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Object)).Xdr(xs)
}
func (v *GETATTR3resok) Xdr(xs *xdr.XdrState) {
	(*Fattr3)(&((v).Obj_attributes)).Xdr(xs)
}
func (v *GETATTR3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*GETATTR3resok)(&((v).Resok)).Xdr(xs)
	default:
	}
}
func (v *Sattrguard3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, (*bool)(&((v).Check)))
	switch (v).Check {
	case true:
		(*Nfstime3)(&((v).Obj_ctime)).Xdr(xs)
	case false:
	}
}
func (v *SETATTR3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Object)).Xdr(xs)
	(*Sattr3)(&((v).New_attributes)).Xdr(xs)
	(*Sattrguard3)(&((v).Guard)).Xdr(xs)
}
func (v *SETATTR3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Obj_wcc)).Xdr(xs)
}
func (v *SETATTR3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Obj_wcc)).Xdr(xs)
}
func (v *SETATTR3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*SETATTR3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*SETATTR3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *LOOKUP3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).What)).Xdr(xs)
}
func (v *LOOKUP3resok) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Object)).Xdr(xs)
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
}
func (v *LOOKUP3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
}
func (v *LOOKUP3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*LOOKUP3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*LOOKUP3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *ACCESS3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Object)).Xdr(xs)
	(*Uint32)(&((v).Access)).Xdr(xs)
}
func (v *ACCESS3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Uint32)(&((v).Access)).Xdr(xs)
}
func (v *ACCESS3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
}
func (v *ACCESS3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*ACCESS3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*ACCESS3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *READLINK3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Symlink)).Xdr(xs)
}
func (v *READLINK3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Symlink_attributes)).Xdr(xs)
	(*Nfspath3)(&((v).Data)).Xdr(xs)
}
func (v *READLINK3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Symlink_attributes)).Xdr(xs)
}
func (v *READLINK3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*READLINK3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*READLINK3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *READ3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).File)).Xdr(xs)
	(*Offset3)(&((v).Offset)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
}
func (v *READ3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).File_attributes)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
	xdr.XdrBool(xs, (*bool)(&((v).Eof)))
	xdr.XdrVarArray(xs, int(-1), (*[]byte)(&((v).Data)))
}
func (v *READ3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).File_attributes)).Xdr(xs)
}
func (v *READ3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*READ3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*READ3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *Stable_how) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *WRITE3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).File)).Xdr(xs)
	(*Offset3)(&((v).Offset)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
	(*Stable_how)(&((v).Stable)).Xdr(xs)
	xdr.XdrVarArray(xs, int(-1), (*[]byte)(&((v).Data)))
}
func (v *WRITE3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).File_wcc)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
	(*Stable_how)(&((v).Committed)).Xdr(xs)
	(*Writeverf3)(&((v).Verf)).Xdr(xs)
}
func (v *WRITE3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).File_wcc)).Xdr(xs)
}
func (v *WRITE3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*WRITE3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*WRITE3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *Createmode3) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}
func (v *Createhow3) Xdr(xs *xdr.XdrState) {
	(*Createmode3)(&((v).Mode)).Xdr(xs)
	switch (v).Mode {
	case UNCHECKED:
		fallthrough
	case GUARDED:
		(*Sattr3)(&((v).Obj_attributes)).Xdr(xs)
	case EXCLUSIVE:
		(*Createverf3)(&((v).Verf)).Xdr(xs)
	}
}
func (v *CREATE3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Where)).Xdr(xs)
	(*Createhow3)(&((v).How)).Xdr(xs)
}
func (v *CREATE3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_fh3)(&((v).Obj)).Xdr(xs)
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *CREATE3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *CREATE3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*CREATE3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*CREATE3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *MKDIR3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Where)).Xdr(xs)
	(*Sattr3)(&((v).Attributes)).Xdr(xs)
}
func (v *MKDIR3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_fh3)(&((v).Obj)).Xdr(xs)
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *MKDIR3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *MKDIR3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*MKDIR3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*MKDIR3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *Symlinkdata3) Xdr(xs *xdr.XdrState) {
	(*Sattr3)(&((v).Symlink_attributes)).Xdr(xs)
	(*Nfspath3)(&((v).Symlink_data)).Xdr(xs)
}
func (v *SYMLINK3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Where)).Xdr(xs)
	(*Symlinkdata3)(&((v).Symlink)).Xdr(xs)
}
func (v *SYMLINK3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_fh3)(&((v).Obj)).Xdr(xs)
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *SYMLINK3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *SYMLINK3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*SYMLINK3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*SYMLINK3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *Devicedata3) Xdr(xs *xdr.XdrState) {
	(*Sattr3)(&((v).Dev_attributes)).Xdr(xs)
	(*Specdata3)(&((v).Spec)).Xdr(xs)
}
func (v *Mknoddata3) Xdr(xs *xdr.XdrState) {
	(*Ftype3)(&((v).Ftype)).Xdr(xs)
	switch (v).Ftype {
	case NF3CHR:
		fallthrough
	case NF3BLK:
		(*Devicedata3)(&((v).Device)).Xdr(xs)
	case NF3SOCK:
		fallthrough
	case NF3FIFO:
		(*Sattr3)(&((v).Pipe_attributes)).Xdr(xs)
	default:
	}
}
func (v *MKNOD3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Where)).Xdr(xs)
	(*Mknoddata3)(&((v).What)).Xdr(xs)
}
func (v *MKNOD3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_fh3)(&((v).Obj)).Xdr(xs)
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *MKNOD3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *MKNOD3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*MKNOD3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*MKNOD3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *REMOVE3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Object)).Xdr(xs)
}
func (v *REMOVE3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *REMOVE3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *REMOVE3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*REMOVE3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*REMOVE3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *RMDIR3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).Object)).Xdr(xs)
}
func (v *RMDIR3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *RMDIR3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Dir_wcc)).Xdr(xs)
}
func (v *RMDIR3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*RMDIR3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*RMDIR3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *RENAME3args) Xdr(xs *xdr.XdrState) {
	(*Diropargs3)(&((v).From)).Xdr(xs)
	(*Diropargs3)(&((v).To)).Xdr(xs)
}
func (v *RENAME3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Fromdir_wcc)).Xdr(xs)
	(*Wcc_data)(&((v).Todir_wcc)).Xdr(xs)
}
func (v *RENAME3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).Fromdir_wcc)).Xdr(xs)
	(*Wcc_data)(&((v).Todir_wcc)).Xdr(xs)
}
func (v *RENAME3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*RENAME3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*RENAME3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *LINK3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).File)).Xdr(xs)
	(*Diropargs3)(&((v).Link)).Xdr(xs)
}
func (v *LINK3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).File_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Linkdir_wcc)).Xdr(xs)
}
func (v *LINK3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).File_attributes)).Xdr(xs)
	(*Wcc_data)(&((v).Linkdir_wcc)).Xdr(xs)
}
func (v *LINK3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*LINK3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*LINK3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *READDIR3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Dir)).Xdr(xs)
	(*Cookie3)(&((v).Cookie)).Xdr(xs)
	(*Cookieverf3)(&((v).Cookieverf)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
}
func (v *Entry3) Xdr(xs *xdr.XdrState) {
	(*Fileid3)(&((v).Fileid)).Xdr(xs)
	(*Filename3)(&((v).Name)).Xdr(xs)
	(*Cookie3)(&((v).Cookie)).Xdr(xs)
	if xs.Encoding() {
		opted := *(&((v).Nextentry)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Entry3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Nextentry)) = new(Entry3)
			(*Entry3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
}
func (v *Dirlist3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&((v).Entries)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Entry3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Entries)) = new(Entry3)
			(*Entry3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	xdr.XdrBool(xs, (*bool)(&((v).Eof)))
}
func (v *READDIR3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
	(*Cookieverf3)(&((v).Cookieverf)).Xdr(xs)
	(*Dirlist3)(&((v).Reply)).Xdr(xs)
}
func (v *READDIR3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
}
func (v *READDIR3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*READDIR3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*READDIR3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *READDIRPLUS3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Dir)).Xdr(xs)
	(*Cookie3)(&((v).Cookie)).Xdr(xs)
	(*Cookieverf3)(&((v).Cookieverf)).Xdr(xs)
	(*Count3)(&((v).Dircount)).Xdr(xs)
	(*Count3)(&((v).Maxcount)).Xdr(xs)
}
func (v *Entryplus3) Xdr(xs *xdr.XdrState) {
	(*Fileid3)(&((v).Fileid)).Xdr(xs)
	(*Filename3)(&((v).Name)).Xdr(xs)
	(*Cookie3)(&((v).Cookie)).Xdr(xs)
	(*Post_op_attr)(&((v).Name_attributes)).Xdr(xs)
	(*Post_op_fh3)(&((v).Name_handle)).Xdr(xs)
	if xs.Encoding() {
		opted := *(&((v).Nextentry)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Entryplus3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Nextentry)) = new(Entryplus3)
			(*Entryplus3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
}
func (v *Dirlistplus3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&((v).Entries)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Entryplus3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Entries)) = new(Entryplus3)
			(*Entryplus3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	xdr.XdrBool(xs, (*bool)(&((v).Eof)))
}
func (v *READDIRPLUS3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
	(*Cookieverf3)(&((v).Cookieverf)).Xdr(xs)
	(*Dirlistplus3)(&((v).Reply)).Xdr(xs)
}
func (v *READDIRPLUS3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Dir_attributes)).Xdr(xs)
}
func (v *READDIRPLUS3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*READDIRPLUS3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*READDIRPLUS3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *FSSTAT3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Fsroot)).Xdr(xs)
}
func (v *FSSTAT3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Size3)(&((v).Tbytes)).Xdr(xs)
	(*Size3)(&((v).Fbytes)).Xdr(xs)
	(*Size3)(&((v).Abytes)).Xdr(xs)
	(*Size3)(&((v).Tfiles)).Xdr(xs)
	(*Size3)(&((v).Ffiles)).Xdr(xs)
	(*Size3)(&((v).Afiles)).Xdr(xs)
	(*Uint32)(&((v).Invarsec)).Xdr(xs)
}
func (v *FSSTAT3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
}
func (v *FSSTAT3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*FSSTAT3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*FSSTAT3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *FSINFO3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Fsroot)).Xdr(xs)
}
func (v *FSINFO3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Uint32)(&((v).Rtmax)).Xdr(xs)
	(*Uint32)(&((v).Rtpref)).Xdr(xs)
	(*Uint32)(&((v).Rtmult)).Xdr(xs)
	(*Uint32)(&((v).Wtmax)).Xdr(xs)
	(*Uint32)(&((v).Wtpref)).Xdr(xs)
	(*Uint32)(&((v).Wtmult)).Xdr(xs)
	(*Uint32)(&((v).Dtpref)).Xdr(xs)
	(*Size3)(&((v).Maxfilesize)).Xdr(xs)
	(*Nfstime3)(&((v).Time_delta)).Xdr(xs)
	(*Uint32)(&((v).Properties)).Xdr(xs)
}
func (v *FSINFO3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
}
func (v *FSINFO3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*FSINFO3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*FSINFO3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *PATHCONF3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).Object)).Xdr(xs)
}
func (v *PATHCONF3resok) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
	(*Uint32)(&((v).Linkmax)).Xdr(xs)
	(*Uint32)(&((v).Name_max)).Xdr(xs)
	xdr.XdrBool(xs, (*bool)(&((v).No_trunc)))
	xdr.XdrBool(xs, (*bool)(&((v).Chown_restricted)))
	xdr.XdrBool(xs, (*bool)(&((v).Case_insensitive)))
	xdr.XdrBool(xs, (*bool)(&((v).Case_preserving)))
}
func (v *PATHCONF3resfail) Xdr(xs *xdr.XdrState) {
	(*Post_op_attr)(&((v).Obj_attributes)).Xdr(xs)
}
func (v *PATHCONF3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*PATHCONF3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*PATHCONF3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *COMMIT3args) Xdr(xs *xdr.XdrState) {
	(*Nfs_fh3)(&((v).File)).Xdr(xs)
	(*Offset3)(&((v).Offset)).Xdr(xs)
	(*Count3)(&((v).Count)).Xdr(xs)
}
func (v *COMMIT3resok) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).File_wcc)).Xdr(xs)
	(*Writeverf3)(&((v).Verf)).Xdr(xs)
}
func (v *COMMIT3resfail) Xdr(xs *xdr.XdrState) {
	(*Wcc_data)(&((v).File_wcc)).Xdr(xs)
}
func (v *COMMIT3res) Xdr(xs *xdr.XdrState) {
	(*Nfsstat3)(&((v).Status)).Xdr(xs)
	switch (v).Status {
	case NFS3_OK:
		(*COMMIT3resok)(&((v).Resok)).Xdr(xs)
	default:
		(*COMMIT3resfail)(&((v).Resfail)).Xdr(xs)
	}
}
func (v *Fhandle3) Xdr(xs *xdr.XdrState) {
	xdr.XdrVarArray(xs, int(FHSIZE3), (*[]byte)(v))
}
func (v *Dirpath3) Xdr(xs *xdr.XdrState) {
	xdr.XdrString(xs, int(MNTPATHLEN3), (*string)(v))
}
func (v *Name3) Xdr(xs *xdr.XdrState) {
	xdr.XdrString(xs, int(MNTNAMLEN3), (*string)(v))
}
func (v *Mountstat3) Xdr(xs *xdr.XdrState) {
	xdr.XdrU32(xs, (*uint32)(v))
}

type MOUNT_PROGRAM_MOUNT_V3_handler interface {
	MOUNTPROC3_NULL()
	MOUNTPROC3_MNT(Dirpath3) Mountres3
	MOUNTPROC3_DUMP() Mountopt3
	MOUNTPROC3_UMNT(Dirpath3)
	MOUNTPROC3_UMNTALL()
	MOUNTPROC3_EXPORT() Exportsopt3
}
type MOUNT_PROGRAM_MOUNT_V3_handler_wrapper struct {
	h MOUNT_PROGRAM_MOUNT_V3_handler
}

func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_NULL(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var out xdr.Void
	w.h.MOUNTPROC3_NULL()
	return &out, nil
}
func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_MNT(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in Dirpath3
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out Mountres3
	out = w.h.MOUNTPROC3_MNT(in)
	return &out, nil
}
func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_DUMP(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var out Mountopt3
	out = w.h.MOUNTPROC3_DUMP()
	return &out, nil
}
func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_UMNT(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var in Dirpath3
	in.Xdr(args)
	err = args.Error()
	if err != nil {
		return
	}
	var out xdr.Void
	w.h.MOUNTPROC3_UMNT(in)
	return &out, nil
}
func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_UMNTALL(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var out xdr.Void
	w.h.MOUNTPROC3_UMNTALL()
	return &out, nil
}
func (w *MOUNT_PROGRAM_MOUNT_V3_handler_wrapper) MOUNTPROC3_EXPORT(args *xdr.XdrState) (res xdr.Xdrable, err error) {
	var out Exportsopt3
	out = w.h.MOUNTPROC3_EXPORT()
	return &out, nil
}
func MOUNT_PROGRAM_MOUNT_V3_regs(h MOUNT_PROGRAM_MOUNT_V3_handler) []xdr.ProcRegistration {
	w := &MOUNT_PROGRAM_MOUNT_V3_handler_wrapper{h}
	return []xdr.ProcRegistration{
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_NULL,
			Handler: w.MOUNTPROC3_NULL,
		},
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_MNT,
			Handler: w.MOUNTPROC3_MNT,
		},
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_DUMP,
			Handler: w.MOUNTPROC3_DUMP,
		},
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_UMNT,
			Handler: w.MOUNTPROC3_UMNT,
		},
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_UMNTALL,
			Handler: w.MOUNTPROC3_UMNTALL,
		},
		xdr.ProcRegistration{
			Prog:    MOUNT_PROGRAM,
			Vers:    MOUNT_V3,
			Proc:    MOUNTPROC3_EXPORT,
			Handler: w.MOUNTPROC3_EXPORT,
		},
	}
}
func (v *Mountres3_ok) Xdr(xs *xdr.XdrState) {
	(*Fhandle3)(&((v).Fhandle)).Xdr(xs)
	{
		var __arraysz uint32
		xs.EncodingSetSize(&__arraysz, len(*&((v).Auth_flavors)))
		xdr.XdrU32(xs, (*uint32)(&__arraysz))

		if xs.Decoding() {
			*&((v).Auth_flavors) = make([]uint32, __arraysz)
		}
		for i := uint64(0); i < uint64(__arraysz); i++ {
			xdr.XdrU32(xs, (*uint32)(&((*(&((v).Auth_flavors)))[i])))

		}
	}
}
func (v *Mountres3) Xdr(xs *xdr.XdrState) {
	(*Mountstat3)(&((v).Fhs_status)).Xdr(xs)
	switch (v).Fhs_status {
	case MNT3_OK:
		(*Mountres3_ok)(&((v).Mountinfo)).Xdr(xs)
	default:
	}
}
func (v *Mount3) Xdr(xs *xdr.XdrState) {
	(*Name3)(&((v).Ml_hostname)).Xdr(xs)
	(*Dirpath3)(&((v).Ml_directory)).Xdr(xs)
	if xs.Encoding() {
		opted := *(&((v).Ml_next)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Mount3)(*(&((v).Ml_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Ml_next)) = new(Mount3)
			(*Mount3)(*(&((v).Ml_next))).Xdr(xs)
		}
	}
}
func (v *Mountopt3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&v.P) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Mount3)(*(&v.P)).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&v.P) = new(Mount3)
			(*Mount3)(*(&v.P)).Xdr(xs)
		}
	}
}
func (v *Groups3) Xdr(xs *xdr.XdrState) {
	(*Name3)(&((v).Gr_name)).Xdr(xs)
	if xs.Encoding() {
		opted := *(&((v).Gr_next)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Groups3)(*(&((v).Gr_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Gr_next)) = new(Groups3)
			(*Groups3)(*(&((v).Gr_next))).Xdr(xs)
		}
	}
}
func (v *Exports3) Xdr(xs *xdr.XdrState) {
	(*Dirpath3)(&((v).Ex_dir)).Xdr(xs)
	if xs.Encoding() {
		opted := *(&((v).Ex_groups)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Groups3)(*(&((v).Ex_groups))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Ex_groups)) = new(Groups3)
			(*Groups3)(*(&((v).Ex_groups))).Xdr(xs)
		}
	}
	if xs.Encoding() {
		opted := *(&((v).Ex_next)) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Exports3)(*(&((v).Ex_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&((v).Ex_next)) = new(Exports3)
			(*Exports3)(*(&((v).Ex_next))).Xdr(xs)
		}
	}
}
func (v *Exportsopt3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&v.P) != nil
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			(*Exports3)(*(&v.P)).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, (*bool)(&opted))
		if opted {
			*(&v.P) = new(Exports3)
			(*Exports3)(*(&v.P)).Xdr(xs)
		}
	}
}
