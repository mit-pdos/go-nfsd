package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fh"
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

func errRet(op *fstxn.FsTxn, status *nfstypes.Nfsstat3, err nfstypes.Nfsstat3) {
	*status = err
	util.DPrintf(1, "errRet %v", err)
	op.Abort()
}

func commitReply(op *fstxn.FsTxn, status *nfstypes.Nfsstat3) {
	ok := op.Commit()
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
	op := fstxn.Begin(nfs.fsstate)
	ip := op.GetInodeFh(args.Object)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE)
		return reply
	}
	reply.Resok.Obj_attributes = ip.MkFattr()
	commitReply(op, &reply.Status)
	return reply
}

// getShrink may lookup an inode that is shrining. If so, do/help shrinking in a
// transaction and redo lookup after shrinking.
func (nfs *Nfs) getShrink(fh nfstypes.Nfs_fh3) (*fstxn.FsTxn, *inode.Inode, nfstypes.Nfsstat3) {
	var op *fstxn.FsTxn
	var ip *inode.Inode
	var ok bool
	var err = nfstypes.NFS3_OK
	for {
		op = fstxn.Begin(nfs.fsstate)
		ip = op.GetInodeFh(fh)
		if ip == nil {
			return op, ip, nfstypes.NFS3ERR_STALE
		}
		if !ip.IsShrinking() {
			break
		}
		inum := ip.Inum
		util.DPrintf(0, "getShrink: abort to shrink")
		op.Abort()
		ok = nfs.shrinkst.DoShrink(inum)
		op = fstxn.Begin(nfs.fsstate)
		if !ok {
			err = nfstypes.NFS3ERR_SERVERFAULT
			break
		}
		util.DPrintf(1, "getShrink: retry %p\n", op.Atxn.Id())
	}
	return op, ip, err
}

func (nfs *Nfs) NFSPROC3_SETATTR(args nfstypes.SETATTR3args) nfstypes.SETATTR3res {
	var reply nfstypes.SETATTR3res
	var err = nfstypes.NFS3ERR_NOTSUPP

	util.DPrintf(1, "NFS SetAttr %v\n", args)
	op, ip, err := nfs.getShrink(args.Object)
	if err != nfstypes.NFS3_OK {
		errRet(op, &reply.Status, err)
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
		shrink := ip.Resize(op.Atxn, uint64(args.New_attributes.Size.Size))
		if shrink {
			nfs.shrinkst.StartShrinker(ip.Inum)
		}
		err = nfstypes.NFS3_OK
	}
	if args.New_attributes.Atime.Set_it != nfstypes.DONT_CHANGE {
		util.DPrintf(1, "NFS SetAttr Atime %v\n", args)
		if args.New_attributes.Atime.Set_it == nfstypes.SET_TO_CLIENT_TIME {
			ip.Atime = args.New_attributes.Atime.Atime
		} else {
			ip.Atime = inode.NfstimeNow()

		}
		ip.WriteInode(op.Atxn)
		err = nfstypes.NFS3_OK
	}
	if args.New_attributes.Mtime.Set_it != nfstypes.DONT_CHANGE {
		util.DPrintf(1, "NFS SetAttr Mtime %v\n", args)
		if args.New_attributes.Mtime.Set_it == nfstypes.SET_TO_CLIENT_TIME {
			ip.Mtime = args.New_attributes.Mtime.Mtime
		} else {
			ip.Mtime = inode.NfstimeNow()

		}
		ip.WriteInode(op.Atxn)
		err = nfstypes.NFS3_OK
	}
	if err == nfstypes.NFS3_OK {
		reply.Resok.Obj_wcc.After.Attributes_follow = true
		reply.Resok.Obj_wcc.After.Attributes = ip.MkFattr()
		commitReply(op, &reply.Status)
	} else {
		errRet(op, &reply.Status, err)
	}
	return reply
}

func twoInodes(ino1, ino2 *inode.Inode) []*inode.Inode {
	inodes := make([]*inode.Inode, 2)
	inodes[0] = ino1
	inodes[1] = ino2
	return inodes
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
		dip := op.GetInodeFh(dfh)
		if dip == nil {
			util.DPrintf(1, "getInodesLocked stale\n")
			err = nfstypes.NFS3ERR_STALE
			break
		}
		inodes = []*inode.Inode{dip}
		inum, _ := dir.LookupName(dip, op, name)
		if inum == common.NULLINUM {
			util.DPrintf(1, "getInodesLocked noent\n")
			err = nfstypes.NFS3ERR_NOENT
			break
		}
		if inum == dip.Inum {
			ip = dip
		} else {
			if inum < dip.Inum {
				// Abort. Try to lock inodes in order
				op.Abort()
				parent := fh.MakeFh(dfh)
				op = fstxn.Begin(nfs.fsstate)
				inodes = lookupOrdered(op, name, parent, inum)
				if inodes == nil {
					ip = nil
				} else {
					ip = inodes[0]
				}
			} else {
				ip = op.GetInodeLocked(inum)
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
		errRet(op, &reply.Status, err)
		return reply
	}
	fh := fh.Fh{Ino: inodes[0].Inum, Gen: inodes[0].Gen}
	reply.Resok.Object = fh.MakeFh3()
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args nfstypes.ACCESS3args) nfstypes.ACCESS3res {
	var reply nfstypes.ACCESS3res
	util.DPrintf(1, "NFS Access %v\n", args)
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Access = nfstypes.Uint32(nfstypes.ACCESS3_READ | nfstypes.ACCESS3_LOOKUP | nfstypes.ACCESS3_MODIFY | nfstypes.ACCESS3_EXTEND | nfstypes.ACCESS3_DELETE | nfstypes.ACCESS3_EXECUTE)
	return reply
}

func (nfs *Nfs) doRead(fh nfstypes.Nfs_fh3, kind nfstypes.Ftype3, offset, count uint64) (*fstxn.FsTxn, []byte, bool, nfstypes.Nfsstat3) {
	var readCount = count
	op := fstxn.Begin(nfs.fsstate)
	ip := op.GetInodeFh(fh)
	if ip == nil {
		return op, nil, false, nfstypes.NFS3ERR_STALE
	}
	if ip.Kind != kind {
		return op, nil, false, nfstypes.NFS3ERR_INVAL
	}
	if ip.Kind == nfstypes.NF3LNK {
		readCount = ip.Size
	}
	data, eof := ip.Read(op.Atxn, offset, readCount)
	return op, data, eof, nfstypes.NFS3_OK
}

func (nfs *Nfs) NFSPROC3_READ(args nfstypes.READ3args) nfstypes.READ3res {
	var reply nfstypes.READ3res
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	op, data, eof, err := nfs.doRead(args.File, nfstypes.NF3REG,
		uint64(args.Offset), uint64(args.Count))
	if err != nfstypes.NFS3_OK {
		errRet(op, &reply.Status, err)
		return reply
	}
	reply.Resok.Count = nfstypes.Count3(len(data))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
	commitReply(op, &reply.Status)
	return reply
}

// XXX Mtime
func (nfs *Nfs) NFSPROC3_WRITE(args nfstypes.WRITE3args) nfstypes.WRITE3res {
	var reply nfstypes.WRITE3res
	var ok = true

	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)

	op, ip, err := nfs.getShrink(args.File)
	if err != nfstypes.NFS3_OK {
		errRet(op, &reply.Status, err)
		return reply

	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
		return reply
	}
	if uint64(args.Count) >= op.Atxn.Buftxn.LogSzBytes() {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
		return reply
	}
	count, writeOk := ip.Write(op.Atxn, uint64(args.Offset), uint64(args.Count),
		args.Data)
	if !writeOk {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_NOSPC)
		return reply
	}
	// if not supporting unstable writes, upgrade stability
	if args.Stable == nfstypes.UNSTABLE && !nfs.Unstable {
		args.Stable = nfstypes.DATA_SYNC
	}
	if args.Stable == nfstypes.FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = op.Commit()
	} else if args.Stable == nfstypes.DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = op.CommitData()
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
		ok = op.CommitUnstable()
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

// getAlloc is complicated because AllocInode() may return an inode
// that needs to be shrunk, and shrinking runs in its own transaction.
func (nfs *Nfs) getAlloc(op *fstxn.FsTxn, dfh nfstypes.Nfs_fh3, name nfstypes.Filename3, kind nfstypes.Ftype3) (*fstxn.FsTxn, *inode.Inode, *inode.Inode, nfstypes.Nfsstat3) {
	var ip *inode.Inode
	var dip *inode.Inode
	var err = nfstypes.NFS3_OK
	for {
		dip = op.GetInodeFh(dfh)
		if dip == nil {
			err = nfstypes.NFS3ERR_STALE
			break
		}
		inum, _ := dir.LookupName(dip, op, name)
		if inum != common.NULLINUM {
			err = nfstypes.NFS3ERR_EXIST
			break
		}
		ip = op.AllocInode(kind)
		if ip == nil {
			err = nfstypes.NFS3ERR_NOSPC
			break
		}
		if !ip.IsShrinking() {
			break
		}
		util.DPrintf(0, "getAlloc: abort alloc # %v to shrink", ip.Inum)
		inum = ip.Inum
		op.Abort()
		ok := nfs.shrinkst.DoShrink(inum)
		op = fstxn.Begin(nfs.fsstate)
		if !ok {
			err = nfstypes.NFS3ERR_SERVERFAULT
			break
		}

		util.DPrintf(1, "getAlloc: retry %p\n", op.Atxn.Id())
	}
	return op, dip, ip, err
}

func (nfs *Nfs) doDecLink(op *fstxn.FsTxn, ip *inode.Inode) {
	if ip.DecLink(op.Atxn) {
		shrink := ip.Resize(op.Atxn, 0)
		ip.FreeInode(op.Atxn)
		if shrink {
			nfs.shrinkst.StartShrinker(ip.Inum)
		}
	}
}

func (nfs *Nfs) doCreate(dfh nfstypes.Nfs_fh3, name nfstypes.Filename3, kind nfstypes.Ftype3,
	data []byte) (op *fstxn.FsTxn, err nfstypes.Nfsstat3, fattr nfstypes.Fattr3) {
	beginOp := fstxn.Begin(nfs.fsstate)
	var dip, ip *inode.Inode
	op, dip, ip, err = nfs.getAlloc(beginOp, dfh, name, kind)
	if err != nfstypes.NFS3_OK {
		return
	}
	if ip == nil {
		err = nfstypes.NFS3ERR_NOSPC
		return
	}
	if kind == nfstypes.NF3DIR {
		ok := dir.InitDir(ip, op, dip.Inum)
		if !ok {
			nfs.doDecLink(op, ip)
			err = nfstypes.NFS3ERR_NOSPC
			return
		}
		dip.Nlink = dip.Nlink + 1 // for ..
		dip.WriteInode(op.Atxn)
	}
	if kind == nfstypes.NF3LNK {
		_, ok := ip.Write(op.Atxn, uint64(0), uint64(len(data)), data)
		if !ok {
			nfs.doDecLink(op, ip)
			err = nfstypes.NFS3ERR_NOSPC
			return
		}
	}
	ok := dir.AddName(dip, op, ip.Inum, name)
	if !ok {
		nfs.doDecLink(op, ip)
		err = nfstypes.NFS3ERR_IO
		return
	}
	err = nfstypes.NFS3_OK
	fattr = ip.MkFattr()
	return
}

func (nfs *Nfs) NFSPROC3_CREATE(args nfstypes.CREATE3args) nfstypes.CREATE3res {
	var reply nfstypes.CREATE3res
	util.DPrintf(1, "NFS Create %v\n", args)
	// XXX deal with how
	if args.How.Mode == nfstypes.EXCLUSIVE {
		reply.Status = nfstypes.NFS3ERR_NOTSUPP
		return reply
	}
	op, err, fattr := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3REG, nil)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Create %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	reply.Resok.Obj_attributes.Attributes_follow = true
	reply.Resok.Obj_attributes.Attributes = fattr
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_MKDIR(args nfstypes.MKDIR3args) nfstypes.MKDIR3res {
	var reply nfstypes.MKDIR3res

	util.DPrintf(1, "NFS MakeDir %v\n", args)
	op, err, fattr := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3DIR, nil)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Create %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	reply.Resok.Obj_attributes.Attributes_follow = true
	reply.Resok.Obj_attributes.Attributes = fattr
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args nfstypes.SYMLINK3args) nfstypes.SYMLINK3res {
	var reply nfstypes.SYMLINK3res
	util.DPrintf(1, "NFS SymLink %v\n", args)

	data := []byte(args.Symlink.Symlink_data)
	op, err, fattr := nfs.doCreate(args.Where.Dir, args.Where.Name, nfstypes.NF3LNK, data)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "doCreate %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	reply.Resok.Obj_attributes.Attributes_follow = true
	reply.Resok.Obj_attributes.Attributes = fattr

	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args nfstypes.READLINK3args) nfstypes.READLINK3res {
	var reply nfstypes.READLINK3res
	util.DPrintf(1, "NFS ReadLink %v\n", args)
	op, data, _, err := nfs.doRead(args.Symlink, nfstypes.NF3LNK, uint64(0), uint64(0))
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "NFS ReadLink err %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	reply.Resok.Data = nfstypes.Nfspath3(string(data))
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args nfstypes.MKNOD3args) nfstypes.MKNOD3res {
	var reply nfstypes.MKNOD3res
	util.DPrintf(1, "NFS MakeNod %v\n", args)
	reply.Status = nfstypes.NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) doRemove(dfh nfstypes.Nfs_fh3, name nfstypes.Filename3, isdir bool) (*fstxn.FsTxn, nfstypes.Nfsstat3) {
	if dir.IllegalName(name) {
		util.DPrintf(0, "Remove inval name\n")
		return nil, nfstypes.NFS3ERR_INVAL
	}
	op, inodes, err := nfs.getInodesLocked(dfh, name)
	if err != nfstypes.NFS3_OK {
		return op, err
	}
	if isdir && inodes[0].Kind != nfstypes.NF3DIR {
		util.DPrintf(0, "Remove not a directory %v\n", inodes[0].Kind)
		return op, nfstypes.NFS3ERR_INVAL
	}
	if isdir && !dir.IsDirEmpty(inodes[0], op) {
		return op, nfstypes.NFS3ERR_INVAL
	}
	ok := dir.RemName(inodes[1], op, name)
	if !ok {
		util.DPrintf(0, "Remove failed\n")
		return op, nfstypes.NFS3ERR_IO
	}
	nfs.doDecLink(op, inodes[0])
	return op, nfstypes.NFS3_OK
}

func (nfs *Nfs) NFSPROC3_REMOVE(args nfstypes.REMOVE3args) nfstypes.REMOVE3res {
	var reply nfstypes.REMOVE3res
	util.DPrintf(1, "NFS Remove %v\n", args)
	op, err := nfs.doRemove(args.Object.Dir, args.Object.Name, false)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(0, "Remove %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args nfstypes.RMDIR3args) nfstypes.RMDIR3res {
	var reply nfstypes.RMDIR3res
	util.DPrintf(1, "NFS Rmdir %v\n", args)
	op, err := nfs.doRemove(args.Object.Dir, args.Object.Name, true)
	if err != nfstypes.NFS3_OK {
		util.DPrintf(1, "Rmdir %v\n", err)
		errRet(op, &reply.Status, err)
		return reply
	}
	commitReply(op, &reply.Status)
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
	var frominum common.Inum
	var toinum common.Inum
	var success bool = false
	var done bool = false

	for !success {
		op = fstxn.Begin(nfs.fsstate)
		util.DPrintf(1, "NFS Rename %v\n", args)

		toh := fh.MakeFh(args.To.Dir)
		fromh := fh.MakeFh(args.From.Dir)

		if dir.IllegalName(args.From.Name) {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
			done = true
			break
		}

		if fh.Equal(args.From.Dir, args.To.Dir) {
			dipfrom = op.GetInodeFh(args.From.Dir)
			if dipfrom == nil {
				errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE)
				done = true
				break
			}
			dipto = dipfrom
			inodes = []*inode.Inode{dipfrom}
		} else {
			inodes = lockInodes(op, twoInums(fromh.Ino, toh.Ino))
			if inodes == nil {
				errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE)
				done = true
				break
			}
			dipfrom = inodes[0]
			dipto = inodes[1]
		}

		util.DPrintf(1, "from %v to %v\n", dipfrom, dipto)

		frominumLookup, _ := dir.LookupName(dipfrom, op, args.From.Name)
		frominum = frominumLookup
		if frominum == common.NULLINUM {
			errRet(op, &reply.Status, nfstypes.NFS3ERR_NOENT)
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
			op.Commit()
			done = true
			break
		}

		// does to exist?
		if toinum != common.NULLINUM {
			// must lock 3 or 4 inodes in order
			var to *inode.Inode
			var from *inode.Inode
			op.Abort()
			op = fstxn.Begin(nfs.fsstate)
			if dipto != dipfrom {
				inums := make([]common.Inum, 4)
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
				inums := make([]common.Inum, 3)
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
					errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
					done = true
					break
				}
				if to.Kind == nfstypes.NF3DIR && !dir.IsDirEmpty(to, op) {
					errRet(op, &reply.Status, nfstypes.NFS3ERR_NOTEMPTY)
					done = true
					break
				}
				ok := dir.RemName(dipto, op, args.To.Name)
				if !ok {
					errRet(op, &reply.Status, nfstypes.NFS3ERR_IO)
					done = true
					break
				}
				nfs.doDecLink(op, to)
				success = true
			} else { // retry
				op.Abort()
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
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO)
		return reply
	}
	ok1 := dir.AddName(dipto, op, frominum, args.To.Name)
	if !ok1 {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO)
		return reply
	}
	commitReply(op, &reply.Status)
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
	ip := op.GetInodeFh(args.Dir)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE)
		return reply
	}
	if ip.Kind != nfstypes.NF3DIR {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
		return reply
	}
	dirlist := Ls3(ip, op, args.Cookie, args.Dircount)
	reply.Resok.Reply = dirlist
	commitReply(op, &reply.Status)
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
	reply.Resok.Rtmax = 16 * 4096
	reply.Resok.Rtmult = 4096
	reply.Resok.Rtpref = reply.Resok.Rtmax
	reply.Resok.Wtmax = nfstypes.Uint32(op.Atxn.Buftxn.LogSzBytes())
	reply.Resok.Wtpref = 16 * 4096
	reply.Resok.Wtmult = 4096
	reply.Resok.Dtpref = 16 * 4096
	reply.Resok.Maxfilesize = nfstypes.Size3(inode.MaxFileSize())
	reply.Resok.Properties = nfstypes.Uint32(nfstypes.FSF3_HOMOGENEOUS | nfstypes.FSF3_SYMLINK)
	commitReply(op, &reply.Status)
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args nfstypes.PATHCONF3args) nfstypes.PATHCONF3res {
	var reply nfstypes.PATHCONF3res
	util.DPrintf(1, "NFS PathConf %v\n", args)
	reply.Status = nfstypes.NFS3_OK
	reply.Resok.Name_max = nfstypes.Uint32(dir.MAXNAMELEN)
	reply.Resok.No_trunc = true
	reply.Resok.Linkmax = 1
	reply.Resok.Case_preserving = true
	return reply
}

// RFC: forces or flushes data to stable storage that was previously
// written with a WRITE procedure call with the stable field set to
// UNSTABLE.
func (nfs *Nfs) NFSPROC3_COMMIT(args nfstypes.COMMIT3args) nfstypes.COMMIT3res {
	var reply nfstypes.COMMIT3res
	util.DPrintf(1, "NFS Commit %v\n", args)
	op := fstxn.Begin(nfs.fsstate)
	ip := op.GetInodeFh(args.File)
	if ip == nil {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_STALE)
		return reply
	}
	if ip.Kind != nfstypes.NF3REG {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
		return reply
	}
	if uint64(args.Offset)+uint64(args.Count) > ip.Size {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_INVAL)
		return reply
	}
	ok := op.CommitFh()
	if ok {
		reply.Status = nfstypes.NFS3_OK
	} else {
		errRet(op, &reply.Status, nfstypes.NFS3ERR_IO)
	}
	return reply
}
