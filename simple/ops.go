package simple

import (
	"github.com/mit-pdos/go-journal/buftxn"
	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/go-journal/util"
)

func fh2ino(fh3 nfstypes.Nfs_fh3) common.Inum {
	fh := MakeFh(fh3)
	return fh.Ino
}

func rootFattr() nfstypes.Fattr3 {
	return nfstypes.Fattr3{
		Ftype: nfstypes.NF3DIR,
		Mode:  0777,
		Nlink: 1,
		Uid:   nfstypes.Uid3(0),
		Gid:   nfstypes.Gid3(0),
		Size:  nfstypes.Size3(0),
		Used:  nfstypes.Size3(0),
		Rdev: nfstypes.Specdata3{Specdata1: nfstypes.Uint32(0),
			Specdata2: nfstypes.Uint32(0)},
		Fsid:   nfstypes.Uint64(0),
		Fileid: nfstypes.Fileid3(common.ROOTINUM),
		Atime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Mtime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
		Ctime: nfstypes.Nfstime3{Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0)},
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(0, "NFS Null\n")
}

func validInum(inum common.Inum) bool {
	if inum == 0 {
		return false
	}
	if inum == common.ROOTINUM {
		return false
	}
	if inum >= nInode() {
		return false
	}
	return true
}

func NFSPROC3_GETATTR_wp(args nfstypes.GETATTR3args, reply *nfstypes.GETATTR3res, inum common.Inum, txn *buftxn.BufTxn) {
	ip := ReadInode(txn, inum)
	reply.Resok.Obj_attributes = ip.MkFattr()
}

func NFSPROC3_GETATTR_internal(args nfstypes.GETATTR3args, reply *nfstypes.GETATTR3res, inum common.Inum, txn *buftxn.BufTxn) {
	NFSPROC3_GETATTR_wp(args, reply, inum, txn)

	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res
	util.DPrintf(1, "NFS GetAttr %v\n", args)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.Object)

	if inum == common.ROOTINUM {
		reply.Status = nfstypes.NFS3_OK
		reply.Resok.Obj_attributes = rootFattr()
		return reply
	}

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	NFSPROC3_GETATTR_internal(args, &reply, inum, txn)
	nfs.l.Release(inum)
	return reply
}

func NFSPROC3_SETATTR_wp(args nfstypes.SETATTR3args, reply *nfstypes.SETATTR3res, inum common.Inum, txn *buftxn.BufTxn) bool {
	ip := ReadInode(txn, inum)

	var ok bool
	if args.New_attributes.Size.Set_it {
		newsize := uint64(args.New_attributes.Size.Size)
		if ip.Size < newsize {
			data := make([]byte, newsize-ip.Size)
			ip.Write(txn, ip.Size, newsize-ip.Size, data)
			if ip.Size != newsize {
				reply.Status = nfstypes.NFS3ERR_NOSPC
			} else {
				ok = true
			}
		} else {
			ip.Size = newsize
			ip.WriteInode(txn)
			ok = true
		}
	} else {
		ok = true
	}

	return ok
}

func NFSPROC3_SETATTR_internal(args nfstypes.SETATTR3args, reply *nfstypes.SETATTR3res, inum common.Inum, txn *buftxn.BufTxn) {
	ok1 := NFSPROC3_SETATTR_wp(args, reply, inum, txn)
	if !ok1 {
		return
	}

	ok2 := txn.CommitWait(true)
	if ok2 {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res
	util.DPrintf(1, "NFS SetAttr %v\n", args)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.Object)

	util.DPrintf(1, "inum %d %d\n", inum, nInode())

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	NFSPROC3_SETATTR_internal(args, &reply, inum, txn)
	nfs.l.Release(inum)
	return reply
}

// Lookup must lock child inode to find gen number
func (nfs *Nfs) NFSPROC3_LOOKUP(args nfstypes.LOOKUP3args) nfstypes.LOOKUP3res {
	util.DPrintf(1, "NFS Lookup %v\n", args)
	var reply nfstypes.LOOKUP3res

	// The filename must be a single letter.
	// 'A' corresponds to inode 0, etc.
	fn := args.What.Name

	var inum common.Inum
	if fn == "a" {
		inum = 2
	}

	if fn == "b" {
		inum = 3
	}

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_NOENT
		return reply
	}

	fh := Fh{Ino: inum}
	reply.Resok.Object = fh.MakeFh3()
	reply.Status = nfstypes.NFS3_OK
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	util.DPrintf(1, "NFS Access %v\n", args)
	var reply nfstypes.ACCESS3res
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Access = nfstypes.Uint32(nfstypes.ACCESS3_READ | nfstypes.ACCESS3_LOOKUP | nfstypes.ACCESS3_MODIFY | nfstypes.ACCESS3_EXTEND | nfstypes.ACCESS3_DELETE | nfstypes.ACCESS3_EXECUTE)
	return reply
}

func NFSPROC3_READ_wp(args nfstypes.READ3args, reply *nfstypes.READ3res, inum common.Inum, txn *buftxn.BufTxn) {
	ip := ReadInode(txn, inum)
	data, eof := ip.Read(txn, uint64(args.Offset), uint64(args.Count))

	reply.Resok.Count = nfstypes.Count3(uint32(len(data)))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
}

func NFSPROC3_READ_internal(args nfstypes.READ3args, reply *nfstypes.READ3res, inum common.Inum, txn *buftxn.BufTxn) {
	NFSPROC3_READ_wp(args, reply, inum, txn)

	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.File)

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	NFSPROC3_READ_internal(args, &reply, inum, txn)
	nfs.l.Release(inum)
	return reply
}

func NFSPROC3_WRITE_wp(args nfstypes.WRITE3args, reply *nfstypes.WRITE3res, inum common.Inum, txn *buftxn.BufTxn) bool {
	ip := ReadInode(txn, inum)

	count, writeok := ip.Write(txn, uint64(args.Offset), uint64(args.Count), args.Data)
	if !writeok {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
		return false
	}

	reply.Resok.Count = nfstypes.Count3(uint32(count))
	reply.Resok.Committed = nfstypes.FILE_SYNC
	return true
}

func NFSPROC3_WRITE_internal(args nfstypes.WRITE3args, reply *nfstypes.WRITE3res, inum common.Inum, txn *buftxn.BufTxn) {
	ok1 := NFSPROC3_WRITE_wp(args, reply, inum, txn)
	if !ok1 {
		return
	}

	ok2 := txn.CommitWait(true)
	if ok2 {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res
	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.File)

	util.DPrintf(1, "inum %d %d\n", inum, nInode())

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	NFSPROC3_WRITE_internal(args, &reply, inum, txn)
	nfs.l.Release(inum)
	return reply
}

func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	util.DPrintf(1, "NFS Create %v\n", args)
	var reply nfstypes.CREATE3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	util.DPrintf(1, "NFS Mkdir %v\n", args)
	var reply nfstypes.MKDIR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	util.DPrintf(1, "NFS Symlink %v\n", args)
	var reply nfstypes.SYMLINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	util.DPrintf(1, "NFS Readlink %v\n", args)
	var reply nfstypes.READLINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args nfstypes.MKNOD3args) nfstypes.MKNOD3res {
	util.DPrintf(1, "NFS Mknod %v\n", args)
	var reply nfstypes.MKNOD3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_REMOVE(args nfstypes.REMOVE3args) nfstypes.REMOVE3res {
	util.DPrintf(1, "NFS Remove %v\n", args)
	var reply nfstypes.REMOVE3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	util.DPrintf(1, "NFS Rmdir %v\n", args)
	var reply nfstypes.RMDIR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_RENAME(args nfstypes.RENAME3args) nfstypes.RENAME3res {
	util.DPrintf(1, "NFS Rename %v\n", args)
	var reply nfstypes.RENAME3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_LINK(args nfstypes.LINK3args) nfstypes.LINK3res {
	util.DPrintf(1, "NFS Link %v\n", args)
	var reply nfstypes.LINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIR(args nfstypes.READDIR3args) nfstypes.READDIR3res {
	util.DPrintf(1, "NFS Readdir %v\n", args)
	var reply nfstypes.READDIR3res

	e2 := &nfstypes.Entry3{
		Fileid:    nfstypes.Fileid3(3),
		Name:      nfstypes.Filename3("b"),
		Cookie:    nfstypes.Cookie3(1),
		Nextentry: nil,
	}
	e1 := &nfstypes.Entry3{
		Fileid:    nfstypes.Fileid3(2),
		Name:      nfstypes.Filename3("a"),
		Cookie:    nfstypes.Cookie3(0),
		Nextentry: e2,
	}
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Reply = nfstypes.Dirlist3{Entries: e1, Eof: true}
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	util.DPrintf(1, "NFS Readdirplus %v\n", args)
	var reply nfstypes.READDIRPLUS3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args nfstypes.FSSTAT3args) nfstypes.FSSTAT3res {
	util.DPrintf(1, "NFS Fsstat %v\n", args)
	var reply nfstypes.FSSTAT3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSINFO(args nfstypes.FSINFO3args) nfstypes.FSINFO3res {
	util.DPrintf(1, "NFS Fsinfo %v\n", args)
	var reply nfstypes.FSINFO3res
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Wtmax = nfstypes.Uint32(4096)
	reply.Resok.Maxfilesize = nfstypes.Size3(4096)
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args nfstypes.PATHCONF3args) nfstypes.PATHCONF3res {
	util.DPrintf(1, "NFS Pathconf %v\n", args)
	var reply nfstypes.PATHCONF3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_COMMIT(args nfstypes.COMMIT3args) nfstypes.COMMIT3res {
	util.DPrintf(1, "NFS Commit %v\n", args)
	var reply nfstypes.COMMIT3res

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.File)

	if !validInum(inum) {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_IO
	}
	nfs.l.Release(inum)
	return reply
}
