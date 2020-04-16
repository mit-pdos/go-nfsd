package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"

	"github.com/mit-pdos/goose-nfsd/barebones"
)

//
// The general plan for implementing each NFS RPC is as follows: start
// a transaction, acquire locks for inodes, perform the requested
// operation, and commit.  Some RPCs require locks to be acquired
// incrementally, because we don't know which inodes the RPC will
// need.  If locking an inode would violate lock ordering, then the
// transaction aborts, and retries (but this time locking inodes in
// lock order).
//

func parseBnfsstat(err barebones.Nfsstat3) nfstypes.Nfsstat3 {
	return nfstypes.Nfsstat3(err)
}

func mkFh(fh3 nfstypes.Nfs_fh3) barebones.Fh {
	fhOld := fh.MakeFh(fh3)
	return barebones.Fh{Ino: fhOld.Ino, Gen: fhOld.Gen}
}

func mkInodeFh3(ip *inode.Inode) nfstypes.Nfs_fh3 {
	return fh.Fh{
		Ino: ip.Inum,
		Gen: ip.Gen,
	}.MakeFh3()
}

func mkInodeFattr(ip *inode.Inode) nfstypes.Fattr3 {
	return nfstypes.Fattr3{
		Ftype: nfstypes.NF3DIR,
		Mode:  0777,
		Nlink: 1,
		Uid:   nfstypes.Uid3(0),
		Gid:   nfstypes.Gid3(0),
		Size:  nfstypes.Size3(0), // size of file
		Used:  nfstypes.Size3(0), // actual disk space used
		Rdev: nfstypes.Specdata3{
			Specdata1: nfstypes.Uint32(0),
			Specdata2: nfstypes.Uint32(0),
		},
		Fsid:   nfstypes.Uint64(0),
		Fileid: nfstypes.Fileid3(ip.Inum), // this is a unique id per file
		Atime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last accessed
		Mtime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last modified
		Ctime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last time attributes were changed, including writes
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(0, "NFS Null\n")
}

func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res

	util.DPrintf(1, "NFS GetAttr %v\n", args)
	fh := mkFh(args.Object)
	ip, err := nfs.Barebones.OpGetAttr(fh)
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.GETATTR3resok{
		Obj_attributes: mkInodeFattr(ip),
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res

	util.DPrintf(1, "NFS SetAttr %v\n", args)

	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

// Lookup must lock child inode to find gen number
func (nfs *Nfs) NFSPROC3_LOOKUP(args nfstypes.LOOKUP3args) nfstypes.LOOKUP3res {
	var reply nfstypes.LOOKUP3res

	util.DPrintf(1, "NFS Lookup %v\n", args)

	fh := mkFh(args.What.Dir)
	dip, ip, err := nfs.Barebones.OpLookup(fh, string(args.What.Name))
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.LOOKUP3resok{
		Object: mkInodeFh3(ip),
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(ip),
		},
		Dir_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(dip),
		},
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	var reply nfstypes.ACCESS3res

	util.DPrintf(1, "NFS Access %v\n", args)

	fh := mkFh(args.Object)
	ip, err := nfs.Barebones.OpGetAttr(fh)
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.ACCESS3resok{
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(ip),
		},
		Access: nfstypes.Uint32(
			nfstypes.ACCESS3_READ |
			nfstypes.ACCESS3_LOOKUP |
			nfstypes.ACCESS3_MODIFY |
			nfstypes.ACCESS3_EXTEND |
			nfstypes.ACCESS3_DELETE |
			// nfstypes.ACCESS3_EXECUTE |
			0,
		),
	}
	return reply
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res

	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res

	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	var reply nfstypes.CREATE3res

	util.DPrintf(1, "NFS Create %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	var reply nfstypes.MKDIR3res

	util.DPrintf(1, "NFS MakeDir %v\n", args)

	fh := mkFh(args.Where.Dir)
	ip, err := nfs.Barebones.OpMkdir(fh, string(args.Where.Name))
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.MKDIR3resok{
		Obj: nfstypes.Post_op_fh3{
			Handle_follows: true,
			Handle: mkInodeFh3(ip),
		},
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(ip),
		},
		Dir_wcc: nfstypes.Wcc_data{
			Before: nfstypes.Pre_op_attr{
				Attributes_follow: false,
			},
			After: nfstypes.Post_op_attr{
				Attributes_follow: false,
			},
		},
	}
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	var reply nfstypes.SYMLINK3res

	util.DPrintf(1, "NFS SymLink %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	var reply nfstypes.READLINK3res

	util.DPrintf(1, "NFS ReadLink %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args nfstypes.MKNOD3args) nfstypes.MKNOD3res {
	var reply nfstypes.MKNOD3res

	util.DPrintf(1, "NFS MakeNod %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_REMOVE(args nfstypes.REMOVE3args) nfstypes.REMOVE3res {
	var reply nfstypes.REMOVE3res

	util.DPrintf(1, "NFS Remove %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	var reply nfstypes.RMDIR3res

	util.DPrintf(1, "NFS Rmdir %v\n", args)

	fh := mkFh(args.Object.Dir)
	err := nfs.Barebones.OpRmdir(fh, string(args.Object.Name))
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.RMDIR3resok{
		Dir_wcc: nfstypes.Wcc_data{
			Before: nfstypes.Pre_op_attr{
				Attributes_follow: false,
			},
			After: nfstypes.Post_op_attr{
				Attributes_follow: false,
			},
		},
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_RENAME(args nfstypes.RENAME3args) nfstypes.RENAME3res {
	var reply nfstypes.RENAME3res

	util.DPrintf(1, "NFS Rename %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_LINK(args nfstypes.LINK3args) nfstypes.LINK3res {
	var reply nfstypes.LINK3res

	util.DPrintf(1, "NFS Link %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) NFSPROC3_READDIR(args nfstypes.READDIR3args) nfstypes.READDIR3res {
	var reply nfstypes.READDIR3res

	util.DPrintf(1, "NFS ReadDir %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func (nfs *Nfs) makeEntry3(entry barebones.Entry3, nextEntry *nfstypes.Entryplus3) *nfstypes.Entryplus3 {
	return &nfstypes.Entryplus3{
		Fileid: nfstypes.Fileid3(entry.Inode.Inum),
		Name: nfstypes.Filename3(entry.Name),
		Cookie: nfstypes.Cookie3(entry.Cookie), // indicates position in dirlist, used for partial dir listing
		Name_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(entry.Inode),
		},
		Name_handle: nfstypes.Post_op_fh3{
			Handle_follows: true,
			Handle: mkInodeFh3(entry.Inode),
		},
		Nextentry: nextEntry,
	}
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	var reply nfstypes.READDIRPLUS3res

	util.DPrintf(1, "NFS ReadDirPlus %v\n", args)

	fh := mkFh(args.Dir)
	dip, bentries, err := nfs.Barebones.OpReadDirPlus(fh)
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}

	var entries *nfstypes.Entryplus3
	for _, entry := range bentries {
		entries = nfs.makeEntry3(entry, entries)
	}
	reply.Resok = nfstypes.READDIRPLUS3resok{
		Dir_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(dip),
		},
		Cookieverf: nfstypes.Cookieverf3{}, // used to check if cookies are still valid
		Reply: nfstypes.Dirlistplus3{
			Entries: entries,
			Eof: true,
		},
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args nfstypes.FSSTAT3args) nfstypes.FSSTAT3res {
	var reply nfstypes.FSSTAT3res

	util.DPrintf(1, "NFS FsStat %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}

func mkDefaultFattr(isDir bool, fileid uint64) nfstypes.Fattr3 {
	ftype := nfstypes.NF3REG
	if isDir {
		ftype = nfstypes.NF3DIR
	}
	return nfstypes.Fattr3{
		Ftype: ftype, // file or directory
		Mode:  0777,
		Nlink: 1,
		Uid:   nfstypes.Uid3(0),
		Gid:   nfstypes.Gid3(0),
		Size:  nfstypes.Size3(0), // size of file
		Used:  nfstypes.Size3(0), // actual disk space used
		Rdev: nfstypes.Specdata3{
			Specdata1: nfstypes.Uint32(0),
			Specdata2: nfstypes.Uint32(0),
		},
		Fsid:   nfstypes.Uint64(0),
		Fileid: nfstypes.Fileid3(fileid), // this is a unique id per file
		Atime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last accessed
		Mtime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last modified
		Ctime: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(0),
			Nseconds: nfstypes.Uint32(0),
		}, // last time attributes were changed, including writes
	}
}

func (nfs *Nfs) NFSPROC3_FSINFO(args nfstypes.FSINFO3args) nfstypes.FSINFO3res {
	var reply nfstypes.FSINFO3res

	util.DPrintf(1, "NFS FsInfo %v\n", args)
	reply.Status = nfstypes.NFS3_OK
	rwUnit := nfstypes.Uint32(disk.BlockSize)
	reply.Resok = nfstypes.FSINFO3resok{
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(nfs.Barebones.GetRootInode()),
		},
		Rtmax: rwUnit,
		Rtpref: rwUnit,
		Rtmult: rwUnit,
		Wtmax: rwUnit,
		Wtpref: rwUnit,
		Wtmult: rwUnit,
		Dtpref: nfstypes.Uint32(0),
		Maxfilesize: nfstypes.Size3(rwUnit),
		Time_delta: nfstypes.Nfstime3{
			Seconds: nfstypes.Uint32(1),
			Nseconds: nfstypes.Uint32(0),
		}, // { 0, 1 } indicates nanosecond precision
		Properties: 0, // no fancy features
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args nfstypes.PATHCONF3args) nfstypes.PATHCONF3res {
	var reply nfstypes.PATHCONF3res

	util.DPrintf(1, "NFS PathConf %v\n", args)
	fh := mkFh(args.Object)
	ip, err := nfs.Barebones.OpGetAttr(fh)
	reply.Status = parseBnfsstat(err)
	if err != barebones.NFS3_OK {
		return reply
	}
	reply.Resok = nfstypes.PATHCONF3resok{
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: mkInodeFattr(ip),
		},
		Linkmax: 0,
		Name_max: 1, // max filename length
		No_trunc: true, // long names are rejected with an error
		Chown_restricted: true,
		Case_insensitive: false,
		Case_preserving: true,
	}

	return reply
}

// RFC: forces or flushes data to stable storage that was previously
// written with a WRITE procedure call with the stable field set to
// UNSTABLE.
func (nfs *Nfs) NFSPROC3_COMMIT(args nfstypes.COMMIT3args) nfstypes.COMMIT3res {
	var reply nfstypes.COMMIT3res

	util.DPrintf(1, "NFS Commit %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP

	return reply
}
