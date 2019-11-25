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
	xdr.XdrBool(xs, &((v).Attributes_follow))
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
	xdr.XdrBool(xs, &((v).Attributes_follow))
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
	xdr.XdrBool(xs, &((v).Handle_follows))
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
	xdr.XdrBool(xs, &((v).Set_it))
	switch (v).Set_it {
	case true:
		(*Mode3)(&((v).Mode)).Xdr(xs)
	default:
	}
}
func (v *Set_uid3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, &((v).Set_it))
	switch (v).Set_it {
	case true:
		(*Uid3)(&((v).Uid)).Xdr(xs)
	default:
	}
}
func (v *Set_gid3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, &((v).Set_it))
	switch (v).Set_it {
	case true:
		(*Gid3)(&((v).Gid)).Xdr(xs)
	default:
	}
}
func (v *Set_size3) Xdr(xs *xdr.XdrState) {
	xdr.XdrBool(xs, &((v).Set_it))
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
	xdr.XdrBool(xs, &((v).Check))
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
	xdr.XdrBool(xs, &((v).Eof))
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
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Entry3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Nextentry)) = new(Entry3)
			(*Entry3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
}
func (v *Dirlist3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&((v).Entries)) != nil
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Entry3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Entries)) = new(Entry3)
			(*Entry3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	xdr.XdrBool(xs, &((v).Eof))
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
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Entryplus3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Nextentry)) = new(Entryplus3)
			(*Entryplus3)(*(&((v).Nextentry))).Xdr(xs)
		}
	}
}
func (v *Dirlistplus3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&((v).Entries)) != nil
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Entryplus3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Entries)) = new(Entryplus3)
			(*Entryplus3)(*(&((v).Entries))).Xdr(xs)
		}
	}
	xdr.XdrBool(xs, &((v).Eof))
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
	xdr.XdrBool(xs, &((v).No_trunc))
	xdr.XdrBool(xs, &((v).Chown_restricted))
	xdr.XdrBool(xs, &((v).Case_insensitive))
	xdr.XdrBool(xs, &((v).Case_preserving))
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
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Mount3)(*(&((v).Ml_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Ml_next)) = new(Mount3)
			(*Mount3)(*(&((v).Ml_next))).Xdr(xs)
		}
	}
}
func (v *Mountopt3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&v.P) != nil
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Mount3)(*(&v.P)).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
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
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Groups3)(*(&((v).Gr_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
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
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Groups3)(*(&((v).Ex_groups))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Ex_groups)) = new(Groups3)
			(*Groups3)(*(&((v).Ex_groups))).Xdr(xs)
		}
	}
	if xs.Encoding() {
		opted := *(&((v).Ex_next)) != nil
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Exports3)(*(&((v).Ex_next))).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&((v).Ex_next)) = new(Exports3)
			(*Exports3)(*(&((v).Ex_next))).Xdr(xs)
		}
	}
}
func (v *Exportsopt3) Xdr(xs *xdr.XdrState) {
	if xs.Encoding() {
		opted := *(&v.P) != nil
		xdr.XdrBool(xs, &opted)
		if opted {
			(*Exports3)(*(&v.P)).Xdr(xs)
		}
	}
	if xs.Decoding() {
		var opted bool
		xdr.XdrBool(xs, &opted)
		if opted {
			*(&v.P) = new(Exports3)
			(*Exports3)(*(&v.P)).Xdr(xs)
		}
	}
}
