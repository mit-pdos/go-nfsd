package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
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

// global locking
func (nfs *Nfs) glockAcq() {
	nfs.glocks.Acquire(0, 0)
}

func (nfs *Nfs) glockRel() {
	nfs.glocks.Release(0, 0)
}

func (nfs *Nfs) getInode(buftxn *buftxn.BufTxn, inum common.Inum) *inode.Inode {
	if inum >= nfs.fs.NInode() {
		return nil
	}
	addr := nfs.fs.Inum2Addr(inum)
	buf := buftxn.ReadBuf(addr, common.INODESZ*8)
	return inode.Decode(buf, inum)
}

func (nfs *Nfs) writeInode(buftxn *buftxn.BufTxn, ip *inode.Inode) {
	addr := nfs.fs.Inum2Addr(ip.Inum)
	buftxn.OverWrite(addr, common.INODESZ*8, ip.Encode())
}

func (nfs *Nfs) getInodeByFh3(buftxn *buftxn.BufTxn, fh3 nfstypes.Nfs_fh3) (*inode.Inode, nfstypes.Nfsstat3) {
	fh := fh.MakeFh(fh3)
	ip := nfs.getInode(buftxn, fh.Ino)
	if ip == nil {
		return nil, nfstypes.NFS3ERR_BADHANDLE
	}
	if ip.Gen != fh.Gen {
		return nil, nfstypes.NFS3ERR_STALE
	}
	return ip, nfstypes.NFS3_OK
}

func (nfs *Nfs) mkRootFattr() nfstypes.Fattr3 {
	buftxn := buftxn.Begin(nfs.txn)
	ip := nfs.getInode(buftxn, common.ROOTINUM)
	return ip.MkFattr()
}

func (nfs *Nfs) lookupName(dip *inode.Inode, name nfstypes.Filename3) common.Inum {
	if name == "." {
		return dip.Inum
	} else if name == ".." {
		return dip.Parent
	}
	for i := 0; i < len(dip.Contents); i++ {
		if dip.Contents[i] == 0 {
			continue
		}
		if dip.Names[i] == name[0] {
			return dip.Contents[i]
		}
	}
	return 0
}

func (nfs *Nfs) allocInode(buftxn *buftxn.BufTxn, dip *inode.Inode) (*inode.Inode, nfstypes.Nfsstat3) {
	for i := uint64(0); i < uint64(len(nfs.bitmap)) * 8; i++ {
		byteNum := i / 8
		bitNum := i % 8
		if (nfs.bitmap[byteNum] & (1 << bitNum)) == 0 {
			nfs.bitmap[byteNum] |= (1 << bitNum)
			bitaddr := addr.MkBitAddr(i / common.NBITBLOCK, i % common.NBITBLOCK)
			buftxn.OverWrite(bitaddr, 1, []byte{1 << bitNum})
			ip := nfs.getInode(buftxn, i)
			ip.InitInode(i, dip.Inum)
			nfs.writeInode(buftxn, ip)
			return ip, nfstypes.NFS3_OK
		}
	}
	return nil, nfstypes.NFS3ERR_NOSPC
}

func (nfs *Nfs) allocDir(buftxn *buftxn.BufTxn, dip *inode.Inode, name nfstypes.Filename3) (*inode.Inode, nfstypes.Nfsstat3) {
	if len(name) == 0 {
		return nil, nfstypes.NFS3ERR_ACCES
	}
	if len(name) > 1 {
		return nil, nfstypes.NFS3ERR_NAMETOOLONG
	}
	existing := nfs.lookupName(dip, name)
	if existing != 0 {
		return nil, nfstypes.NFS3ERR_EXIST
	}
	for i := 0; i < len(dip.Contents); i++ {
		if dip.Contents[i] == 0 {
			ip, err := nfs.allocInode(buftxn, dip)
			if err != nfstypes.NFS3_OK {
				return nil, err
			}
			dip.Contents[i] = ip.Inum
			dip.Names[i] = name[0]
			nfs.writeInode(buftxn, dip)
			return ip, err
		}
	}
	return nil, nfstypes.NFS3ERR_ACCES
}

func (nfs *Nfs) freeRecurse(buftxn *buftxn.BufTxn, dip *inode.Inode) {
	for i := 0; i < len(dip.Contents); i++ {
		if dip.Contents[i] == 0{
			continue
		}
		ip := nfs.getInode(buftxn, dip.Contents[i])
		nfs.freeRecurse(buftxn, ip)
	}
	byteNum := dip.Inum / 8
	bitNum := dip.Inum % 8
	nfs.bitmap[byteNum] &= ^(1 << bitNum)
	bitaddr := addr.MkBitAddr(dip.Inum / common.NBITBLOCK, dip.Inum % common.NBITBLOCK)
	buftxn.OverWrite(bitaddr, 1, []byte{ 0 })
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(0, "NFS Null\n")
}

func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res

	util.DPrintf(1, "NFS GetAttr %v\n", args)
	buftxn := buftxn.Begin(nfs.txn)
	ip, err := nfs.getInodeByFh3(buftxn, args.Object)
	if err != nfstypes.NFS3_OK {
		reply.Status = err
		return reply
	}
	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.GETATTR3resok{
		Obj_attributes: ip.MkFattr(),
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

	buftxn := buftxn.Begin(nfs.txn)

	dip, err1 := nfs.getInodeByFh3(buftxn, args.What.Dir)
	if err1 != nfstypes.NFS3_OK {
		reply.Status = err1
		nfs.glockRel()
		return reply
	}
	in := nfs.lookupName(dip, args.What.Name)
	if in == 0 {
		reply.Status = nfstypes.NFS3ERR_NOENT
		nfs.glockRel()
		return reply
	}
	ip := nfs.getInode(buftxn, in)
	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.LOOKUP3resok{
		Object: ip.MkFh3(),
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: ip.MkFattr(),
		},
		Dir_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: dip.MkFattr(),
		},
	}

	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	var reply nfstypes.ACCESS3res

	util.DPrintf(1, "NFS Access %v\n", args)

	buftxn := buftxn.Begin(nfs.txn)
	ip, err := nfs.getInodeByFh3(buftxn, args.Object)
	if err != nfstypes.NFS3_OK {
		reply.Status = err
		return reply
	}
	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.ACCESS3resok{
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: ip.MkFattr(),
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

	buftxn := buftxn.Begin(nfs.txn)
	nfs.glockAcq()

	dip, err1 := nfs.getInodeByFh3(buftxn, args.Where.Dir)
	if err1 != nfstypes.NFS3_OK {
		reply.Status = err1
		nfs.glockRel()
		return reply
	}
	ip, err2 := nfs.allocDir(buftxn, dip, args.Where.Name)
	if err2 != nfstypes.NFS3_OK {
		reply.Status = err2
		nfs.glockRel()
		return reply
	}

	buftxn.CommitWait(false)
	nfs.glockRel()

	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.MKDIR3resok{
		Obj: nfstypes.Post_op_fh3{
			Handle_follows: true,
			Handle: fh.Fh{
				Ino: ip.Inum,
				Gen: ip.Gen,
			}.MakeFh3(),
		},
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: ip.MkFattr(),
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

	if args.Object.Name == "." {
		reply.Status = nfstypes.NFS3ERR_INVAL
		return reply
	}
	if args.Object.Name == ".." {
		reply.Status = nfstypes.NFS3ERR_EXIST
		return reply
	}

	buftxn := buftxn.Begin(nfs.txn)
	nfs.glockAcq()

	dip, err1 := nfs.getInodeByFh3(buftxn, args.Object.Dir)
	if err1 != nfstypes.NFS3_OK {
		reply.Status = err1
		nfs.glockRel()
		return reply
	}
	in := nfs.lookupName(dip, args.Object.Name)
	if in == 0 {
		reply.Status = nfstypes.NFS3ERR_NOENT
		nfs.glockRel()
		return reply
	}
	ip := nfs.getInode(buftxn, in)
	nfs.freeRecurse(buftxn, ip)
	for i := 0; i < len(dip.Contents); i++ {
		if dip.Contents[i] == in {
			dip.Contents[i] = 0
			break
		}
	}
	nfs.writeInode(buftxn, dip)

	buftxn.CommitWait(false)
	nfs.glockRel()

	reply.Status = nfstypes.NFS3_OK
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

func (nfs *Nfs) makeEntry3(ip *inode.Inode, name []byte, cookie uint64, nextEntry *nfstypes.Entryplus3) *nfstypes.Entryplus3 {
	return &nfstypes.Entryplus3{
		Fileid: nfstypes.Fileid3(ip.Inum),
		Name: nfstypes.Filename3(name),
		Cookie: nfstypes.Cookie3(cookie), // indicates position in dirlist, used for partial dir listing
		Name_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: ip.MkFattr(),
		},
		Name_handle: nfstypes.Post_op_fh3{
			Handle_follows: true,
			Handle: ip.MkFh3(),
		},
		Nextentry: nextEntry,
	}
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	var reply nfstypes.READDIRPLUS3res

	util.DPrintf(1, "NFS ReadDirPlus %v\n", args)
	buftxn := buftxn.Begin(nfs.txn)
	dip, err := nfs.getInodeByFh3(buftxn, args.Dir)
	if err != nfstypes.NFS3_OK {
		reply.Status = err
		return reply
	}
	entries := nfs.makeEntry3(dip, []byte("."), 1, nil)
	if dip.Parent != 0 {
		pip := nfs.getInode(buftxn, dip.Parent)
		entries = nfs.makeEntry3(pip, []byte(".."), 2, entries)
	}
	for ir := uint64(0); ir < uint64(len(dip.Contents)); ir++ {
		i := uint64(len(dip.Contents)) - ir - 1
		if dip.Contents[i] == 0 {
			continue
		}
		ip := nfs.getInode(buftxn, dip.Contents[i])
		entries = nfs.makeEntry3(ip, []byte{ dip.Names[i] }, i + 3, entries)
	}
	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.READDIRPLUS3resok{
		Dir_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: dip.MkFattr(),
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
			Attributes: nfs.mkRootFattr(),
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
	reply.Status = nfstypes.NFS3_OK
	reply.Resok = nfstypes.PATHCONF3resok{
		Obj_attributes: nfstypes.Post_op_attr{
			Attributes_follow: true,
			Attributes: nfs.mkRootFattr(),
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
