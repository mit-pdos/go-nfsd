package simple

import (
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
)

type Nfs struct {
	t *txn.Txn
	s *super.FsSuper
	l *lockmap.LockMap
}

func fh2ino(fh3 nfstypes.Nfs_fh3) common.Inum {
	fh := fh.MakeFh(fh3)
	return fh.Ino
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(0, "NFS Null\n")
}

func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res
	util.DPrintf(1, "NFS GetAttr %v\n", args)

	// XXX: special case for directory: common.ROOTINUM

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.Object)

	if inum >= nfs.s.NInode() {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	ip := ReadInode(txn, nfs.s, inum)
	reply.Resok.Obj_attributes = ip.MkFattr()
	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}

	nfs.l.Release(inum)
	return reply
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

// Lookup must lock child inode to find gen number
func (nfs *Nfs) NFSPROC3_LOOKUP(args nfstypes.LOOKUP3args) nfstypes.LOOKUP3res {
	var reply nfstypes.LOOKUP3res

	// XXX look up in top-level directory

	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	var reply nfstypes.ACCESS3res
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Access = nfstypes.Uint32(nfstypes.ACCESS3_READ | nfstypes.ACCESS3_LOOKUP | nfstypes.ACCESS3_MODIFY | nfstypes.ACCESS3_EXTEND | nfstypes.ACCESS3_DELETE | nfstypes.ACCESS3_EXECUTE)
	return reply
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.File)

	if inum >= nfs.s.NInode() {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	ip := ReadInode(txn, nfs.s, inum)

	data, eof := ip.Read(txn, nfs.s, uint64(args.Offset), uint64(args.Count))

	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
		reply.Resok.Count = nfstypes.Count3(len(data))
		reply.Resok.Data = data
		reply.Resok.Eof = eof
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}

	nfs.l.Release(inum)
	return reply
}

func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res
	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)

	txn := buftxn.Begin(nfs.t)
	inum := fh2ino(args.File)

	if inum >= nfs.s.NInode() {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}

	nfs.l.Acquire(inum)
	ip := ReadInode(txn, nfs.s, inum)

	count, writeok := ip.Write(txn, nfs.s, uint64(args.Offset), uint64(args.Count), args.Data)
	if !writeok {
		nfs.l.Release(inum)
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
		return reply
	}

	var ok bool
	if args.Stable == nfstypes.FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = txn.CommitWait(true)
	} else if args.Stable == nfstypes.DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = txn.CommitWait(true)
	} else {
		// RFC:	"UNSTABLE, the server is free to commit
		// any part of the data and the metadata to
		// stable storage, including all or none,
		// before returning a reply to the
		// client. There is no guarantee whether or
		// when any uncommitted data will subsequently
		// be committed to stable storage. The only
		// guarantees made by the server are that it
		// will not destroy any data without changing
		// the value of verf and that it will not
		// commit the data and metadata at a level
		// less than that requested by the client."
		ok = txn.CommitWait(false)
	}

	if ok {
		reply.Status = nfstypes.NFS3_OK
		reply.Resok.Count = nfstypes.Count3(count)
		reply.Resok.Committed = args.Stable
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}

	nfs.l.Release(inum)
	return reply
}

func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	var reply nfstypes.CREATE3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	var reply nfstypes.MKDIR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	var reply nfstypes.SYMLINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	var reply nfstypes.READLINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args nfstypes.MKNOD3args) nfstypes.MKNOD3res {
	var reply nfstypes.MKNOD3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_REMOVE(args nfstypes.REMOVE3args) nfstypes.REMOVE3res {
	var reply nfstypes.REMOVE3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	var reply nfstypes.RMDIR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_RENAME(args nfstypes.RENAME3args) nfstypes.RENAME3res {
	var reply nfstypes.RENAME3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_LINK(args nfstypes.LINK3args) nfstypes.LINK3res {
	var reply nfstypes.LINK3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIR(args nfstypes.READDIR3args) nfstypes.READDIR3res {
	var reply nfstypes.READDIR3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	var reply nfstypes.READDIRPLUS3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args nfstypes.FSSTAT3args) nfstypes.FSSTAT3res {
	var reply nfstypes.FSSTAT3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSINFO(args nfstypes.FSINFO3args) nfstypes.FSINFO3res {
	var reply nfstypes.FSINFO3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args nfstypes.PATHCONF3args) nfstypes.PATHCONF3res {
	var reply nfstypes.PATHCONF3res
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_COMMIT(args nfstypes.COMMIT3args) nfstypes.COMMIT3res {
	var reply nfstypes.COMMIT3res
	txn := buftxn.Begin(nfs.t)
	ok := txn.CommitWait(true)
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		reply.Status = nfstypes.NFS3ERR_IO
	}
	return reply
}
