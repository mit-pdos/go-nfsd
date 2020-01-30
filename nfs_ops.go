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

//
// The general plan for implementing each NFS RPC is as follows: start
// a transaction, acquire locks for inodes, perform the requested
// operation, and commit.  Some RPCs require locks to be acquired
// incrementally, because we don't know which inodes the RPC will
// need.  If locking an inode would violate lock ordering, then the
// transaction aborts, and retries (but this time locking inodes in
// lock order).
//

func twoInodes(ino1, ino2 *inode.Inode) []*inode.Inode {
	inodes := make([]*inode.Inode, 2)
	inodes[0] = ino1
	inodes[1] = ino2
	return inodes
}

func errRet(op *fstxn.FsTxn, status *nfstypes.Nfsstat3, err nfstypes.Nfsstat3,
	inodes []*inode.Inode) {
	*status = err
	util.DPrintf(1, "errRet %v", err)
	inode.Abort(op, inodes)
}

func commitReply(op *fstxn.FsTxn, status *nfstypes.Nfsstat3, inodes []*inode.Inode) {
	ok := inode.Commit(op, inodes)
	if ok {
		*status = nfstypes.NFS3_OK
	} else {
		*status = nfstypes.NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(0, "NFS Null\n")
}

func (nfs *Nfs) NFSPROC3_GETATTR(args nfstypes.GETATTR3args) nfstypes.GETATTR3res {
	var reply nfstypes.GETATTR3res
	util.DPrintf(1, "NFS GetAttr %v\n", args)
	txn := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInodeFh(txn, args.Object)
	if ip == nil {
		errRet(txn, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	reply.Resok.Obj_attributes = ip.MkFattr()
	commitReply(txn, &reply.Status, inode.OneInode(ip))
	return reply
}

// If caller changes file size and shrinking is in progress (because
// an earlier call truncated the file), then help/wait with/for
// shrinking.
func (nfs *Nfs) helpShrinker(op *fstxn.FsTxn, ip *inode.Inode,
	fh nfstypes.Nfs_fh3) (*inode.Inode, bool) {
	var ok bool = true
	for ip.IsShrinking() {
		ip.Shrink(op)
		ok = inode.Commit(op, inode.OneInode(ip))
		if !ok {
			break
		}
		op = fstxn.Begin(nfs.fsstate)
		ip = inode.GetInodeFh(op, fh)
	}
	return ip, ok
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res
	var err = nfstypes.NFS3ERR_NOTSUPP

	util.DPrintf(1, "NFS SetAttr %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInodeFh(op, args.Object)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if args.New_attributes.Mode.Set_it {
		util.DPrintf(1, "NFS SetAttr ignore mode %v\n", args)
		err = nfstypes.NFS3_OK
	}
	if args.New_attributes.Uid.Set_it {
		util.DPrintf(1, "NFS SetAttr uid not supported %v\n", args)
	}
	if args.New_attributes.Gid.Set_it {
		util.DPrintf(1, "NFS SetAttr gid not supported %v\n", args)
	}
	if args.New_attributes.Size.Set_it {
		ip, ok := nfs.helpShrinker(op, ip, args.Object)
		if ok {
			ip.Resize(op, uint64(args.New_attributes.Size.Size))
			err = nfstypes.NFS3_OK
		} else {
			err = nfstypes.NFS3ERR_SERVERFAULT
		}
	}
	if args.New_attributes.Atime.Set_it != nfstypes.DONT_CHANGE {
		util.DPrintf(1, "NFS SetAttr Atime %v\n", args)
		if args.New_attributes.Atime.Set_it == nfstypes.SET_TO_CLIENT_TIME {
			ip.Atime = args.New_attributes.Atime.Atime
		} else {
			ip.Atime = inode.NfstimeNow()

		}
		ip.WriteInode(op)
		err = nfstypes.NFS3_OK
	}
	if args.New_attributes.Mtime.Set_it != nfstypes.DONT_CHANGE {
		util.DPrintf(1, "NFS SetAttr Mtime %v\n", args)
		if args.New_attributes.Mtime.Set_it == nfstypes.SET_TO_CLIENT_TIME {
			ip.Mtime = args.New_attributes.Mtime.Mtime
		} else {
			ip.Mtime = inode.NfstimeNow()

		}
		ip.WriteInode(op)
		err = nfstypes.NFS3_OK
	}
	if err == nfstypes.NFS3_OK {
		commitReply(op, &reply.Status, inode.OneInode(ip))
	} else {
		errRet(op, &reply.Status, err, inode.OneInode(ip))
	}
	return reply
}

// Lock the inode for dfh and the inode for name.  name may be a
// directory (e.g., "."). We must lock directories in ascending inum
// order.
func (nfs *Nfs) getInodesLocked(dfh nfstypes.Nfs_fh3, name nfstypes.Filename3) (*fstxn.FsTxn, []*inode.Inode, nfstypes.Nfsstat3) {
	var err nfstypes.Nfsstat3 = nfstypes.NFS3_OK
	var inodes []*inode.Inode
	var ip *inode.Inode
	var op *fstxn.FsTxn

	for ip == nil {
		op = fstxn.Begin(nfs.fsstate)
		util.DPrintf(1, "getInodesLocked %v %v\n", dfh, name)
		dip := inode.GetInodeFh(op, dfh)
		if dip == nil {
			util.DPrintf(1, "getInodesLocked stale\n")
			err = nfstypes.NFS3ERR_STALE
			break
		}
		inodes = inode.OneInode(dip)
		inum, _ := dir.LookupName(dip, op, name)
		if inum == fs.NULLINUM {
			util.DPrintf(1, "getInodesLocked noent\n")
			err = nfstypes.NFS3ERR_NOENT
			break
		}
		if inum == dip.Inum {
			ip = dip
		} else {
			if inum < dip.Inum {
				// Abort. Try to lock inodes in order
				inode.Abort(op, inode.OneInode(dip))
				parent := fh.MakeFh(dfh)
				op = fstxn.Begin(nfs.fsstate)
				inodes = lookupOrdered(op, name, parent, inum)
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
	return op, inodes, err
}

// Lookup must lock child inode to find gen number
func (nfs *Nfs) NFSPROC3_LOOKUP(args nfstypes.LOOKUP3args) nfstypes.LOOKUP3res {
	var reply nfstypes.LOOKUP3res

	util.DPrintf(1, "NFS Lookup %v\n", args)
	op, inodes, err := nfs.getInodesLocked(args.What.Dir, args.What.Name)
	if err != nfstypes.NFS3_OK {
		errRet(op, &reply.Status, err, inodes)
		return reply
	}
	fh := fh.Fh{Ino: inodes[0].Inum, Gen: inodes[0].Gen}
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

func (nfs *Nfs) doRead(fh nfstypes.Nfs_fh3, kind nfstypes.Ftype3, offset, count uint64) (*fstxn.FsTxn, []*inode.Inode, []byte, bool, nfstypes.Nfsstat3) {
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInodeFh(op, fh)
	if ip == nil {
		return op, nil, nil, false, nfstypes.NFS3ERR_STALE
	}
	if ip.Kind != kind {
		return op, inode.OneInode(ip), nil, false, nfstypes.NFS3ERR_INVAL
	}
	if ip.Kind == nfstypes.NF3LNK {
		count = ip.Size
	}
	data, eof := ip.Read(op, offset, count)
	return op, inode.OneInode(ip), data, eof, nfstypes.NFS3_OK
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	op, inode, data, eof, err := nfs.doRead(args.File, nfstypes.NF3REG,
		uint64(args.Offset), uint64(args.Count))
	if err != nfstypes.NFS3_OK {
		errRet(op, &reply.Status, err, inode)
		return reply
	}
	reply.Resok.Count = nfstypes.Count3(len(data))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
	commitReply(op, &reply.Status, inode)
	return reply
}

// XXX Mtime
func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res
	var ok = true

	op := fstxn.Begin(nfs.fsstate)
	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)
	ip := inode.GetInodeFh(op, args.File)
	fh := fh.MakeFh(args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inode.OneInode(ip))
		return reply
	}
	if uint64(args.Count) >= op.LogSzBytes() {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inode.OneInode(ip))
		return reply
	}

	// XXX only when this RPC might grow file?
	ip, ok = nfs.helpShrinker(op, ip, args.File)
	if !ok {
		reply.Status = nfstypes.NFS3ERR_SERVERFAULT
		return reply
	}

	count, writeOk := ip.Write(op, uint64(args.Offset), uint64(args.Count),
		args.Data)
	if !writeOk {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC, inode.OneInode(ip))
		return reply
	}
	if args.Stable == nfstypes.FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = inode.Commit(op, inode.OneInode(ip))
	} else if args.Stable == nfstypes.DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = inode.CommitData(op, inode.OneInode(ip), fh)
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
		ok = inode.CommitUnstable(op, inode.OneInode(ip), fh)
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

func (nfs *Nfs) doCreate(dfh nfstypes.Nfs_fh3, name nfstypes.Filename3, kind nfstypes.Ftype3, data []byte) (*fstxn.FsTxn, []*inode.Inode, nfstypes.Nfsstat3) {
	op := fstxn.Begin(nfs.fsstate)
	dip := inode.GetInodeFh(op, dfh)
	if dip == nil {
		return op, nil, nfstypes.NFS3ERR_STALE
	}
	inum1, _ := dir.LookupName(dip, op, name)
	if inum1 != fs.NULLINUM {
		return op, inode.OneInode(dip), nfstypes.NFS3ERR_EXIST
	}
	inum, ip := inode.AllocInode(op, kind)
	if inum == fs.NULLINUM {
		return op, inode.OneInode(dip), nfstypes.NFS3ERR_NOSPC
	}
	if kind == nfstypes.NF3DIR {
		ok := dir.InitDir(ip, op, dip.Inum)
		if !ok {
			ip.DecLink(op)
			return op, twoInodes(dip, ip), nfstypes.NFS3ERR_NOSPC
		}
		dip.Nlink = dip.Nlink + 1 // for ..
		dip.WriteInode(op)
	}
	if kind == nfstypes.NF3LNK {
		_, ok := ip.Write(op, uint64(0), uint64(len(data)), data)
		if !ok {
			ip.DecLink(op)
			return op, twoInodes(dip, ip), nfstypes.NFS3ERR_NOSPC
		}
	}
	ok := dir.AddName(dip, op, inum, name)
	if !ok {
		ip.DecLink(op)
		return op, twoInodes(dip, ip), nfstypes.NFS3ERR_IO
	}
	return op, twoInodes(dip, ip), nfstypes.NFS3_OK
}

// XXX deal with how
func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	var reply nfstypes.CREATE3res
	util.DPrintf(1, "NFS Create %v\n", args)
	op, inodes, err := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3REG, nil)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Create %v\n", err)
		errRet(op, &reply.Status, err, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	var reply nfstypes.MKDIR3res

	util.DPrintf(1, "NFS MakeDir %v\n", args)
	op, inodes, err := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3DIR, nil)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Create %v\n", err)
		errRet(op, &reply.Status, err, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	var reply nfstypes.SYMLINK3res
	util.DPrintf(1, "NFS SymLink %v\n", args)

	data := []byte(args.Symlink.Symlink_data)
	op, inodes, err := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3LNK, data)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "doCreate %v\n", err)
		errRet(op, &reply.Status, err, inodes)
		return reply
	}

	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	var reply nfstypes.READLINK3res
	util.DPrintf(1, "NFS ReadLink %v\n", args)
	op, inode, data, _, err := nfs.doRead(args.Symlink, nfstypes.NF3LNK,
		uint64(0), uint64(0))
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "NFS ReadLink err %v\n", err)
		errRet(op, &reply.Status, err, inode)
		return reply
	}
	reply.Resok.Data = nfstypes.Nfspath3(string(data))
	commitReply(op, &reply.Status, inode)
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args nfstypes.MKNOD3args) nfstypes.MKNOD3res {
	var reply nfstypes.MKNOD3res
	util.DPrintf(1, "NFS MakeNod %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) doRemove(dfh nfstypes.Nfs_fh3, name nfstypes.Filename3, isdir bool) (*fstxn.FsTxn, []*inode.Inode, nfstypes.Nfsstat3) {
	if dir.IllegalName(name) {
		util.DPrintf(0, "Remove inval name\n")
		return nil, []*inode.Inode{}, nfstypes.NFS3ERR_INVAL
	}
	op, inodes, err := nfs.getInodesLocked(dfh, name)
	if err != nfstypes.NFS3_OK {
		return op, inodes, err
	}
	if isdir && inodes[0].Kind != nfstypes.NF3DIR {
		util.DPrintf(0, "Remove not a directory %v\n", inodes[0].Kind)
		return op, inodes, nfstypes.NFS3ERR_INVAL
	}
	if isdir && !dir.IsDirEmpty(inodes[0], op) {
		return op, inodes, nfstypes.NFS3ERR_INVAL
	}
	ok := dir.RemName(inodes[1], op, name)
	if !ok {
		util.DPrintf(0, "Remove failed\n")
		return op, inodes, nfstypes.NFS3ERR_IO
	}
	inodes[0].DecLink(op)
	return op, inodes, nfstypes.NFS3_OK
}

func (nfs *Nfs) NFSPROC3_REMOVE(args nfstypes.REMOVE3args) nfstypes.REMOVE3res {
	var reply nfstypes.REMOVE3res
	util.DPrintf(1, "NFS Remove %v\n", args)
	op, inodes, err := nfs.doRemove(args.Object.Dir, args.Object.Name, false)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(0, "Remove %v\n", err)
		errRet(op, &reply.Status, err, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	var reply nfstypes.RMDIR3res
	util.DPrintf(1, "NFS Rmdir %v\n", args)
	op, inodes, err := nfs.doRemove(args.Object.Dir, args.Object.Name, true)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Rmdir %v\n", err)
		errRet(op, &reply.Status, err, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
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
			dipfrom = inode.GetInodeFh(op, args.From.Dir)
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

		util.DPrintf(1, "from %v to %v\n", dipfrom, dipto)

		frominumLookup, _ := dir.LookupName(dipfrom, op, args.From.Name)
		frominum = frominumLookup
		if frominum == fs.NULLINUM {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_NOENT, inodes)
			done = true
			break
		}
		util.DPrintf(1, "frominum %d toinum %d\n", frominum, toinum)

		toInumLookup, _ := dir.LookupName(dipto, op, args.To.Name)
		toinum = toInumLookup

		util.DPrintf(1, "frominum %d toinum %d\n", frominum, toinum)

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
			util.DPrintf(1, "inodes %v\n", inodes)
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
	util.DPrintf(1, "NFS ReadDir %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args nfstypes.READDIRPLUS3args) nfstypes.READDIRPLUS3res {
	var reply nfstypes.READDIRPLUS3res
	util.DPrintf(1, "NFS ReadDirPlus %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	ip := inode.GetInodeFh(op, args.Dir)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	inodes := inode.OneInode(ip)
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
	ip := inode.GetInodeFh(op, args.File)
	fh := fh.MakeFh(args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE, nil)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inode.OneInode(ip))
		return reply
	}
	if uint64(args.Offset)+uint64(args.Count) > ip.Size {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL, inode.OneInode(ip))
		return reply
	}
	ok := inode.CommitFh(op, fh, inode.OneInode(ip))
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO, inode.OneInode(ip))
	}
	return reply
}
