package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

func errRet(op *fstxn.FsTxn, status *nfstypes.Nfsstat3, err nfstypes.Nfsstat3,
	inodes []*inode.Inode) {
	*status = err
	util.DPrintf(1, "errRet %v", err)
	inode.Abort(op, inodes)
}

func commitReply(op *fstxn.FsTxn, status *nfstypes.Nfsstat3, inodes []*inode.Inode) {
	ok := inode.Commit(op, inodes)
	util.DPrintf(1, "commitReply %v %v", ok, status)
	if ok {
		*status = nfstypes.NFS3_OK
	} else {
		*status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(1, "NFS Null\n")
}

// XXX factor out lookup ip, test, and maybe fail pattern
func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res
	util.DPrintf(1, "NFS GetAttr %v\n", args)
	txn := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInode(txn, args.Object)
	if ip == nil {
		errRet(txn, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	reply.Resok.Obj_attributes = ip.MkFattr()
	commitReply(txn, &reply.Status, []*inode.Inode{ip})
	return reply
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res
	util.DPrintf(1, "NFS SetAttr %v\n", args)
	trans := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInode(trans, args.Object)
	if ip == nil {
		errRet(trans, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if args.New_attributes.Size.Set_it {
		ip.Resize(trans, uint64(args.New_attributes.Size.Size))
		commitReply(trans, &reply.Status, []*inode.Inode{ip})
	} else {
		errRet(trans, &reply.Status, nfstypes.NFS3ERR_NOTSUPP, []*inode.Inode{ip})
	}
	return reply
}

// Lookup must lock child inode to find gen number, but child maybe a
// directory. We must lock directories in ascending inum order.
func (nfs *Nfs) NFSPROC3_LOOKUP(args nfstypes.LOOKUP3args) nfstypes.LOOKUP3res {
	var reply nfstypes.LOOKUP3res
	var ip *inode.Inode
	var inodes []*inode.Inode
	var op *fstxn.FsTxn
	var done bool = false
	for ip == nil {
		op = fstxn.Begin(nfs.fsstate)
		util.DPrintf(1, "NFS Lookup %v\n", args)
		dip := inode.GetInode(op, args.What.Dir)
		if dip == nil {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
			done = true
			break
		}
		inum, _ := dir.LookupName(dip, op, args.What.Name)
		if inum == fs.NULLINUM {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_NOENT, []*inode.Inode{dip})
			done = true
			break
		}
		inodes = []*inode.Inode{dip}
		if inum == dip.Inum {
			ip = dip
		} else {
			if inum < dip.Inum {
				// Abort. Try to lock inodes in order
				inode.Abort(op, []*inode.Inode{dip})
				parent := fh.MakeFh(args.What.Dir)
				op = fstxn.Begin(nfs.fsstate)
				inodes = lookupOrdered(op, args.What.Name, parent, inum)
				if inodes == nil {
					ip = nil
				} else {
					ip = inodes[0]
				}
			} else {
				ip = inode.GetInodeLocked(op, inum)
				inodes = twoInodes(ip, dip)
			}
		}
	}
	if done {
		return reply
	}
	fh := fh.Fh{Ino: ip.Inum, Gen: ip.Gen}
	reply.Resok.Object = fh.MakeFh3()
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	var reply nfstypes.ACCESS3res
	util.DPrintf(1, "NFS Access %v\n", args)
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Access = nfstypes.Uint32(nfstypes.ACCESS3_READ | nfstypes.ACCESS3_LOOKUP | nfstypes.ACCESS3_MODIFY | nfstypes.ACCESS3_EXTEND | nfstypes.ACCESS3_DELETE | nfstypes.ACCESS3_EXECUTE)
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	var reply nfstypes.READLINK3res
	util.DPrintf(1, "NFS ReadLink %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res
	op := fstxn.Begin(nfs.fsstate)
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	ip := inode.GetInode(op, args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{ip})
		return reply
	}
	data, eof := ip.Read(op, uint64(args.Offset), uint64(args.Count))
	reply.Resok.Count = nfstypes.Count3(len(data))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
	commitReply(op, &reply.Status, []*inode.Inode{ip})
	return reply
}

// XXX Mtime
func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res
	op := fstxn.Begin(nfs.fsstate)
	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)
	ip := inode.GetInode(op, args.File)
	fh := fh.MakeFh(args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{ip})
		return reply
	}
	if uint64(args.Count) >= op.LogSzBytes() {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{ip})
		return reply
	}
	count, writeOk := ip.Write(op, uint64(args.Offset), uint64(args.Count),
		args.Data)
	if !writeOk {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC, []*inode.Inode{ip})
		return reply
	}
	var ok = true
	if args.Stable == nfstypes.FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = inode.Commit(op, []*inode.Inode{ip})
	} else if args.Stable == nfstypes.DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = inode.CommitData(op, []*inode.Inode{ip}, fh)
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
		ok = inode.CommitUnstable(op, []*inode.Inode{ip}, fh)
	}
	if ok {
		reply.Status = nfstypes.NFS3_OK
		reply.Resok.Count = nfstypes.Count3(count)
		reply.Resok.Committed = args.Stable
	} else {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
	}
	return reply
}

// XXX deal with how
func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	var reply nfstypes.CREATE3res
	op := fstxn.Begin(nfs.fsstate)
	util.DPrintf(1, "NFS Create %v\n", args)
	dip := inode.GetInode(op, args.Where.Dir)
	if dip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dir.LookupName(dip, op, args.Where.Name)
	if inum1 != fs.NULLINUM {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_EXIST, []*inode.Inode{dip})
		return reply
	}
	inum, ip := inode.AllocInode(op, nfstypes.NF3REG)
	if inum == fs.NULLINUM {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC, []*inode.Inode{dip})
		return reply
	}
	ok := dir.AddName(dip, op, inum, args.Where.Name)
	if !ok {
		inode.FreeInum(op, inum)
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, []*inode.Inode{dip})
		return reply
	}
	commitReply(op, &reply.Status, []*inode.Inode{dip, ip})
	return reply
}

func twoInodes(ino1, ino2 *inode.Inode) []*inode.Inode {
	inodes := make([]*inode.Inode, 2)
	inodes[0] = ino1
	inodes[1] = ino2
	return inodes
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	var reply nfstypes.MKDIR3res
	op := fstxn.Begin(nfs.fsstate)
	util.DPrintf(1, "NFS MakeDir %v\n", args)
	dip := inode.GetInode(op, args.Where.Dir)
	if dip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dir.LookupName(dip, op, args.Where.Name)
	if inum1 != fs.NULLINUM {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_EXIST, []*inode.Inode{dip})
		return reply
	}
	inum, ip := inode.AllocInode(op, nfstypes.NF3DIR)
	if inum == fs.NULLINUM {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC, []*inode.Inode{dip})
		return reply
	}
	ok := dir.InitDir(ip, op, dip.Inum)
	if !ok {
		ip.DecLink(op)
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC, twoInodes(dip, ip))
		return reply
	}
	ok1 := dir.AddName(dip, op, inum, args.Where.Name)
	if !ok1 {
		ip.DecLink(op)
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, twoInodes(dip, ip))
		return reply
	}
	dip.Nlink = dip.Nlink + 1 // for ..
	dip.WriteInode(op)
	commitReply(op, &reply.Status, twoInodes(dip, ip))
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	var reply nfstypes.SYMLINK3res
	util.DPrintf(1, "NFS MakeDir %v\n", args)
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
	var ip *inode.Inode
	var dip *inode.Inode
	var inodes []*inode.Inode
	var op *fstxn.FsTxn
	var done bool = false
	for ip == nil {
		op = fstxn.Begin(nfs.fsstate)
		util.DPrintf(1, "NFS Remove %v\n", args)
		dip = inode.GetInode(op, args.Object.Dir)
		if dip == nil {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
			done = true
			break
		}
		if dir.IllegalName(args.Object.Name) {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{dip})
			done = true
			break
		}
		inum, _ := dir.LookupName(dip, op, args.Object.Name)
		if inum == fs.NULLINUM {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_NOENT, []*inode.Inode{dip})
			done = true
			break
		}
		if inum < dip.Inum {
			// Abort. Try to lock inodes in order
			inode.Abort(op, []*inode.Inode{dip})
			op := fstxn.Begin(nfs.fsstate)
			parent := fh.MakeFh(args.Object.Dir)
			inodes = lookupOrdered(op, args.Object.Name, parent, inum)
			ip = inodes[0]
			dip = inodes[1]
		} else {
			ip = inode.GetInodeLocked(op, inum)
			inodes = twoInodes(ip, dip)
		}
	}
	if done {
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inodes)
		return reply
	}
	ok := dir.RemName(dip, op, args.Object.Name)
	if !ok {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, inodes)
		return reply
	}
	ip.DecLink(op)
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	var reply nfstypes.RMDIR3res
	util.DPrintf(1, "NFS RmDir %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func validateRename(op *fstxn.FsTxn, inodes []*inode.Inode, fromfh fh.Fh, tofh fh.Fh,
	fromn nfstypes.Filename3, ton nfstypes.Filename3) bool {
	var dipto *inode.Inode
	var dipfrom *inode.Inode
	var from *inode.Inode
	var to *inode.Inode
	if len(inodes) == 3 {
		dipfrom = inodes[0]
		dipto = inodes[0]
		from = inodes[1]
		to = inodes[2]
	} else {
		dipfrom = inodes[0]
		dipto = inodes[1]
		from = inodes[2]
		to = inodes[3]
	}
	if dipfrom.Inum != fromfh.Ino || dipfrom.Gen != fromfh.Gen ||
		dipto.Inum != tofh.Ino || dipto.Gen != tofh.Gen {
		util.DPrintf(10, "revalidate ino failed\n")
		return false
	}
	frominum, _ := dir.LookupName(dipfrom, op, fromn)
	toinum, _ := dir.LookupName(dipto, op, ton)
	if from.Inum != frominum || toinum != to.Inum {
		util.DPrintf(10, "revalidate inums failed\n")
		return false
	}
	return true
}

func (nfs *Nfs) NFSPROC3_RENAME(args nfstypes.RENAME3args) nfstypes.RENAME3res {
	var reply nfstypes.RENAME3res
	var dipto *inode.Inode
	var dipfrom *inode.Inode
	var op *fstxn.FsTxn
	var inodes []*inode.Inode
	var frominum fs.Inum
	var toinum fs.Inum
	var success bool = false
	var done bool = false

	for !success {
		op = fstxn.Begin(nfs.fsstate)
		util.DPrintf(1, "NFS Rename %v\n", args)

		toh := fh.MakeFh(args.To.Dir)
		fromh := fh.MakeFh(args.From.Dir)

		if dir.IllegalName(args.From.Name) {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, nil)
			done = true
			break
		}

		if fh.Equal(args.From.Dir, args.To.Dir) {
			dipfrom = inode.GetInode(op, args.From.Dir)
			if dipfrom == nil {
				errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
				done = true
				break
			}
			dipto = dipfrom
			inodes = []*inode.Inode{dipfrom}
		} else {
			inodes = lockInodes(op, twoInums(fromh.Ino, toh.Ino))
			if inodes == nil {
				errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, inodes)
				done = true
				break
			}
			dipfrom = inodes[0]
			dipto = inodes[1]
		}

		util.DPrintf(5, "from %v to %v\n", dipfrom, dipto)

		frominumLookup, _ := dir.LookupName(dipfrom, op, args.From.Name)
		frominum = frominumLookup
		if frominum == fs.NULLINUM {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_NOENT, inodes)
			done = true
			break
		}

		toInumLookup, _ := dir.LookupName(dipto, op, args.To.Name)
		toinum = toInumLookup

		util.DPrintf(5, "frominum %d toinum %d\n", frominum, toinum)

		// rename to itself?
		if dipto == dipfrom && toinum == frominum {
			reply.Status = nfstypes.NFS3_OK
			inode.Commit(op, inodes)
			done = true
			break
		}

		// does to exist?
		if toinum != fs.NULLINUM {
			// must lock 3 or 4 inodes in order
			var to *inode.Inode
			var from *inode.Inode
			inode.Abort(op, inodes)
			op = fstxn.Begin(nfs.fsstate)
			if dipto != dipfrom {
				inums := make([]fs.Inum, 4)
				inums[0] = dipfrom.Inum
				inums[1] = dipto.Inum
				inums[2] = frominum
				inums[3] = toinum
				inodes = lockInodes(op, inums)
				dipfrom = inodes[0]
				dipto = inodes[1]
				from = inodes[2]
				to = inodes[3]
			} else {
				inums := make([]fs.Inum, 3)
				inums[0] = dipfrom.Inum
				inums[1] = frominum
				inums[2] = toinum
				inodes = lockInodes(op, inums)
				dipfrom = inodes[0]
				dipto = inodes[0]
				from = inodes[1]
				to = inodes[2]
			}
			util.DPrintf(5, "inodes %v\n", inodes)
			if validateRename(op, inodes, fromh, toh,
				args.From.Name, args.To.Name) {
				if to.Kind != from.Kind {
					errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inodes)
					done = true
					break
				}
				if to.Kind == nfstypes.NF3DIR && !dir.IsDirEmpty(to, op) {
					errRet(op, &reply.Status, nfstypes.NFS3ERR_NOTEMPTY, inodes)
					done = true
					break
				}
				ok := dir.RemName(dipto, op, args.To.Name)
				if !ok {
					errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, inodes)
					done = true
					break
				}
				to.DecLink(op)
				success = true
			} else { // retry
				inode.Abort(op, inodes)
			}
		} else {
			success = true
		}
	}
	if done {
		return reply
	}
	ok := dir.RemName(dipfrom, op, args.From.Name)
	if !ok {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, inodes)
		return reply
	}
	ok1 := dir.AddName(dipto, op, frominum, args.To.Name)
	if !ok1 {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
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
	util.DPrintf(1, "NFS Link %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	var reply nfstypes.READDIRPLUS3res
	util.DPrintf(1, "NFS ReadDirPlus %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInode(op, args.Dir)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	inodes := []*inode.Inode{ip}
	if ip.Kind != nfstypes.NF3DIR {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inodes)
		return reply
	}
	dirlist := Ls3(ip, op, args.Cookie, args.Dircount)
	reply.Resok.Reply = dirlist
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args nfstypes.FSSTAT3args) nfstypes.FSSTAT3res {
	var reply nfstypes.FSSTAT3res
	util.DPrintf(1, "NFS FsStat %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSINFO(args nfstypes.FSINFO3args) nfstypes.FSINFO3res {
	var reply nfstypes.FSINFO3res
	util.DPrintf(1, "NFS FsInfo %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	reply.Resok.Wtmax = nfstypes.Uint32(op.LogSzBytes())
	reply.Resok.Maxfilesize = nfstypes.Size3(inode.MaxFileSize())
	commitReply(op, &reply.Status, []*inode.Inode{})
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args nfstypes.PATHCONF3args) nfstypes.PATHCONF3res {
	var reply nfstypes.PATHCONF3res
	util.DPrintf(1, "NFS PathConf %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

// RFC: forces or flushes data to stable storage that was previously
// written with a WRITE procedure call with the stable field set to
// UNSTABLE.
func (nfs *Nfs) NFSPROC3_COMMIT(args nfstypes.COMMIT3args) nfstypes.COMMIT3res {
	var reply nfstypes.COMMIT3res
	util.DPrintf(1, "NFS Commit %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInode(op, args.File)
	fh := fh.MakeFh(args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{ip})
		return reply
	}
	if uint64(args.Offset)+uint64(args.Count) > ip.Size {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, []*inode.Inode{ip})
		return reply
	}
	ok := inode.CommitFh(op, fh, []*inode.Inode{ip})
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, []*inode.Inode{ip})
	}
	return reply
}
