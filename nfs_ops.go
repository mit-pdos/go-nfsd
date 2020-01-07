package goose_nfs

import (
	"sort"
	"sync"

	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/trans"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/util"
	"github.com/mit-pdos/goose-nfsd/wal"
)

const ICACHESZ uint64 = 20               // XXX resurrect icache
const BCACHESZ uint64 = fs.HDRADDRS + 10 // At least as big as log

type Nfs struct {
	mu       *sync.Mutex
	condShut *sync.Cond
	txn      *txn.Txn
	fs       *fs.FsSuper
	balloc   *trans.Alloc
	ialloc   *trans.Alloc
	nthread  uint32
}

var nfs *Nfs

// XXX call recovery, once nfs uses persistent storage
func MkNfs() *Nfs {
	super := fs.MkFsSuper() // run first so that disk is initialized before mkLog
	util.DPrintf(1, "Super: %v\n", super)

	l := wal.MkLog()
	if l == nil {
		panic("mkLog failed")
	}

	initFs(super)

	mu := new(sync.Mutex)
	nfs = &Nfs{
		mu:       mu,
		condShut: sync.NewCond(mu),
		txn:      txn.MkTxn(super),
		fs:       super,
		balloc:   trans.MkAlloc(super.BitmapBlockStart(), super.NBlockBitmap),
		ialloc:   trans.MkAlloc(super.BitmapInodeStart(), super.NInodeBitmap),
		nthread:  0,
	}
	nfs.makeRootDir()
	return nfs
}

func (nfs *Nfs) makeRootDir() {
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	ip := getInodeInum(trans, ROOTINUM)
	if ip == nil {
		panic("makeRootDir")
	}
	ip.mkRootDir(trans)
	ok := commit(trans, []*inode{ip})
	if !ok {
		panic("makeRootDir")
	}
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.mu.Lock()
	for nfs.nthread > 0 {
		util.DPrintf(1, "ShutdownNfs: wait %d\n", nfs.nthread)
		nfs.condShut.Wait()
	}
	nfs.mu.Unlock()
	nfs.txn.Shutdown()
}

func commitWait(trans *trans.Trans, inodes []*inode, wait bool, abort bool) bool {
	// putInodes may free an inode so must be done before commit
	putInodes(trans, inodes)
	return trans.CommitWait(wait, abort)
}

func commit(trans *trans.Trans, inodes []*inode) bool {
	return commitWait(trans, inodes, true, false)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func commitData(trans *trans.Trans, inodes []*inode, fh fh) bool {
	return commitWait(trans, inodes, true, false)
}

// Commit transaction, but don't write to stable storage
func commitUnstable(trans *trans.Trans, inodes []*inode, fh fh) bool {
	util.DPrintf(5, "commitUnstable\n")
	if len(inodes) > 1 {
		panic("commitUnstable")
	}
	return commitWait(trans, inodes, false, false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func commitFh(trans *trans.Trans, fh fh, inodes []*inode) bool {
	return trans.Flush()
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func abort(trans *trans.Trans, inodes []*inode) bool {
	util.DPrintf(5, "Abort\n")
	return commitWait(trans, inodes, true, true)
}

func errRet(trans *trans.Trans, status *Nfsstat3, err Nfsstat3, inodes []*inode) {
	*status = err
	abort(trans, inodes)
}

func commitReply(trans *trans.Trans, status *Nfsstat3, inodes []*inode) {
	ok := commit(trans, inodes)
	if ok {
		*status = NFS3_OK
	} else {
		*status = NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	util.DPrintf(1, "NFS Null\n")
}

// XXX factor out lookup ip, test, and maybe fail pattern
func (nfs *Nfs) NFSPROC3_GETATTR(args GETATTR3args) GETATTR3res {
	var reply GETATTR3res
	util.DPrintf(1, "NFS GetAttr %v\n", args)
	txn := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	ip := getInode(txn, args.Object)
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	reply.Resok.Obj_attributes = ip.mkFattr()
	commitReply(txn, &reply.Status, []*inode{ip})
	return reply
}

func (nfs *Nfs) NFSPROC3_SETATTR(args SETATTR3args) SETATTR3res {
	var reply SETATTR3res
	util.DPrintf(1, "NFS SetAttr %v\n", args)
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	ip := getInode(trans, args.Object)
	if ip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if args.New_attributes.Size.Set_it {
		ip.resize(trans, uint64(args.New_attributes.Size.Size))
		commitReply(trans, &reply.Status, []*inode{ip})
	} else {
		errRet(trans, &reply.Status, NFS3ERR_NOTSUPP, []*inode{ip})
	}
	return reply
}

// Lock inodes in sorted order, but return the pointers in the same order as in inums
// Caller must revalidate inodes.
func lockInodes(trans *trans.Trans, inums []fs.Inum) []*inode {
	util.DPrintf(1, "lock inodes %v\n", inums)
	sorted := make([]fs.Inum, len(inums))
	copy(sorted, inums)
	sort.Slice(sorted, func(i, j int) bool { return inums[i] < inums[j] })
	var inodes = make([]*inode, len(inums))
	for _, inm := range sorted {
		ip := getInodeInum(trans, inm)
		if ip == nil {
			abort(trans, inodes)
			return nil
		}
		// put in same position as in inums
		pos := func(inm fs.Inum) int {
			for i, v := range inums {
				if v == inm {
					return i
				}
			}
			panic("func")
		}(inm)
		inodes[pos] = ip
	}
	return inodes
}

func twoInums(inum1, inum2 fs.Inum) []fs.Inum {
	inums := make([]fs.Inum, 2)
	inums[0] = inum1
	inums[1] = inum2
	return inums
}

// First lookup inode up for child, then for parent, because parent
// inum > child inum and then revalidate that child is still in parent
// directory.
func (nfs *Nfs) lookupOrdered(trans *trans.Trans, name Filename3, parent fh, inm fs.Inum) []*inode {
	util.DPrintf(5, "NFS lookupOrdered child %d parent %v\n", inm, parent)
	inodes := lockInodes(trans, twoInums(inm, parent.ino))
	if inodes == nil {
		return nil
	}
	dip := inodes[1]
	if dip.gen != parent.gen {
		abort(trans, inodes)
		return nil
	}
	child, _ := dip.lookupName(trans, name)
	if child == NULLINUM || child != inm {
		abort(trans, inodes)
		return nil
	}
	return inodes
}

// Lookup must lock child inode to find gen number, but child maybe a
// directory. We must lock directories in ascending inum order.
func (nfs *Nfs) NFSPROC3_LOOKUP(args LOOKUP3args) LOOKUP3res {
	var reply LOOKUP3res
	var ip *inode
	var inodes []*inode
	var op *trans.Trans
	var done bool = false
	for ip == nil {
		op = trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
		util.DPrintf(1, "NFS Lookup %v\n", args)
		dip := getInode(op, args.What.Dir)
		if dip == nil {
			errRet(op, &reply.Status, NFS3ERR_STALE, nil)
			done = true
			break
		}
		inum, _ := dip.lookupName(op, args.What.Name)
		if inum == NULLINUM {
			errRet(op, &reply.Status, NFS3ERR_NOENT, []*inode{dip})
			done = true
			break
		}
		inodes = []*inode{dip}
		if inum == dip.inum {
			ip = dip
		} else {
			if inum < dip.inum {
				// Abort. Try to lock inodes in order
				abort(op, []*inode{dip})
				parent := args.What.Dir.makeFh()
				op = trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
				inodes = nfs.lookupOrdered(op, args.What.Name,
					parent, inum)
				if inodes == nil {
					ip = nil
				} else {
					ip = inodes[0]
				}
			} else {
				ip = getInodeLocked(op, inum)
				inodes = twoInodes(ip, dip)
			}
		}
	}
	if done {
		return reply
	}
	fh := fh{ino: ip.inum, gen: ip.gen}
	reply.Resok.Object = fh.makeFh3()
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args ACCESS3args) ACCESS3res {
	var reply ACCESS3res
	util.DPrintf(1, "NFS Access %v\n", args)
	reply.Status = NFS3_OK
	reply.Resok.Access = Uint32(ACCESS3_READ | ACCESS3_LOOKUP | ACCESS3_MODIFY | ACCESS3_EXTEND | ACCESS3_DELETE | ACCESS3_EXECUTE)
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args READLINK3args) READLINK3res {
	var reply READLINK3res
	util.DPrintf(1, "NFS ReadLink %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READ(args READ3args) READ3res {
	var reply READ3res
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	util.DPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	ip := getInode(trans, args.File)
	if ip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	data, eof := ip.read(trans, uint64(args.Offset), uint64(args.Count))
	reply.Resok.Count = Count3(len(data))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
	commitReply(trans, &reply.Status, []*inode{ip})
	return reply
}

// XXX Mtime
func (nfs *Nfs) NFSPROC3_WRITE(args WRITE3args) WRITE3res {
	var reply WRITE3res
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	util.DPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)
	ip := getInode(trans, args.File)
	fh := args.File.makeFh()
	if ip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	if uint64(args.Count) >= trans.LogSzBytes() {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	count, writeOk := ip.write(trans, uint64(args.Offset), uint64(args.Count),
		args.Data)
	if !writeOk {
		errRet(trans, &reply.Status, NFS3ERR_NOSPC, []*inode{ip})
		return reply
	}
	var ok = true
	if args.Stable == FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = commit(trans, []*inode{ip})
	} else if args.Stable == DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = commitData(trans, []*inode{ip}, fh)
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
		ok = commitUnstable(trans, []*inode{ip}, fh)
	}
	if ok {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(count)
		reply.Resok.Committed = args.Stable
	} else {
		reply.Status = NFS3ERR_SERVERFAULT
	}
	return reply
}

// XXX deal with how
func (nfs *Nfs) NFSPROC3_CREATE(args CREATE3args) CREATE3res {
	var reply CREATE3res
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	util.DPrintf(1, "NFS Create %v\n", args)
	dip := getInode(trans, args.Where.Dir)
	if dip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dip.lookupName(trans, args.Where.Name)
	if inum1 != NULLINUM {
		errRet(trans, &reply.Status, NFS3ERR_EXIST, []*inode{dip})
		return reply
	}
	inum := allocInode(trans, NF3REG)
	if inum == NULLINUM {
		errRet(trans, &reply.Status, NFS3ERR_NOSPC, []*inode{dip})
		return reply
	}
	ok := dip.addName(trans, inum, args.Where.Name)
	if !ok {
		freeInum(trans, inum)
		errRet(trans, &reply.Status, NFS3ERR_IO, []*inode{dip})
		return reply
	}
	commitReply(trans, &reply.Status, []*inode{dip})
	return reply
}

func twoInodes(ino1, ino2 *inode) []*inode {
	inodes := make([]*inode, 2)
	inodes[0] = ino1
	inodes[1] = ino2
	return inodes
}

func (nfs *Nfs) NFSPROC3_MKDIR(args MKDIR3args) MKDIR3res {
	var reply MKDIR3res
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	util.DPrintf(1, "NFS MakeDir %v\n", args)
	dip := getInode(trans, args.Where.Dir)
	if dip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dip.lookupName(trans, args.Where.Name)
	if inum1 != NULLINUM {
		errRet(trans, &reply.Status, NFS3ERR_EXIST, []*inode{dip})
		return reply
	}
	inum := allocInode(trans, NF3DIR)
	if inum == NULLINUM {
		errRet(trans, &reply.Status, NFS3ERR_NOSPC, []*inode{dip})
		return reply
	}
	ip := getInodeLocked(trans, inum)
	ok := ip.initDir(trans, dip.inum)
	if !ok {
		ip.decLink(trans)
		errRet(trans, &reply.Status, NFS3ERR_NOSPC, twoInodes(dip, ip))
		return reply
	}
	ok1 := dip.addName(trans, inum, args.Where.Name)
	if !ok1 {
		ip.decLink(trans)
		errRet(trans, &reply.Status, NFS3ERR_IO, twoInodes(dip, ip))
		return reply
	}
	dip.nlink = dip.nlink + 1 // for ..
	dip.writeInode(trans)
	commitReply(trans, &reply.Status, twoInodes(dip, ip))
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args SYMLINK3args) SYMLINK3res {
	var reply SYMLINK3res
	util.DPrintf(1, "NFS MakeDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args MKNOD3args) MKNOD3res {
	var reply MKNOD3res
	util.DPrintf(1, "NFS MakeNod %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_REMOVE(args REMOVE3args) REMOVE3res {
	var reply REMOVE3res
	var ip *inode
	var dip *inode
	var inodes []*inode
	var op *trans.Trans
	var done bool = false
	for ip == nil {
		op = trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
		util.DPrintf(1, "NFS Remove %v\n", args)
		dip = getInode(op, args.Object.Dir)
		if dip == nil {
			errRet(op, &reply.Status, NFS3ERR_STALE, nil)
			done = true
			break
		}
		if illegalName(args.Object.Name) {
			errRet(op, &reply.Status, NFS3ERR_INVAL, []*inode{dip})
			done = true
			break
		}
		inum, _ := dip.lookupName(op, args.Object.Name)
		if inum == NULLINUM {
			errRet(op, &reply.Status, NFS3ERR_NOENT, []*inode{dip})
			done = true
			break
		}
		if inum < dip.inum {
			// Abort. Try to lock inodes in order
			abort(op, []*inode{dip})
			op := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
			parent := args.Object.Dir.makeFh()
			inodes = nfs.lookupOrdered(op, args.Object.Name, parent, inum)
			ip = inodes[0]
			dip = inodes[1]
		} else {
			ip = getInodeLocked(op, inum)
			inodes = twoInodes(ip, dip)
		}
	}
	if done {
		return reply
	}
	if ip.kind != NF3REG {
		errRet(op, &reply.Status, NFS3ERR_INVAL, inodes)
		return reply
	}
	ok := dip.remName(op, args.Object.Name)
	if !ok {
		errRet(op, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	ip.decLink(op)
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args RMDIR3args) RMDIR3res {
	var reply RMDIR3res
	util.DPrintf(1, "NFS RmDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func validateRename(trans *trans.Trans, inodes []*inode, fromfh fh, tofh fh,
	fromn Filename3, ton Filename3) bool {
	var dipto *inode
	var dipfrom *inode
	var from *inode
	var to *inode
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
	if dipfrom.inum != fromfh.ino || dipfrom.gen != fromfh.gen ||
		dipto.inum != tofh.ino || dipto.gen != tofh.gen {
		util.DPrintf(10, "revalidate ino failed\n")
		return false
	}
	frominum, _ := dipfrom.lookupName(trans, fromn)
	toinum, _ := dipto.lookupName(trans, ton)
	if from.inum != frominum || toinum != to.inum {
		util.DPrintf(10, "revalidate inums failed\n")
		return false
	}
	return true
}

func (nfs *Nfs) NFSPROC3_RENAME(args RENAME3args) RENAME3res {
	var reply RENAME3res
	var dipto *inode
	var dipfrom *inode
	var op *trans.Trans
	var inodes []*inode
	var frominum fs.Inum
	var toinum fs.Inum
	var success bool = false
	var done bool = false

	for !success {
		op = trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
		util.DPrintf(1, "NFS Rename %v\n", args)

		toh := args.To.Dir.makeFh()
		fromh := args.From.Dir.makeFh()

		if illegalName(args.From.Name) {
			errRet(op, &reply.Status, NFS3ERR_INVAL, nil)
			done = true
			break
		}

		if args.From.Dir.equal(args.To.Dir) {
			dipfrom = getInode(op, args.From.Dir)
			if dipfrom == nil {
				errRet(op, &reply.Status, NFS3ERR_STALE, nil)
				done = true
				break
			}
			dipto = dipfrom
			inodes = []*inode{dipfrom}
		} else {
			inodes = lockInodes(op, twoInums(fromh.ino, toh.ino))
			if inodes == nil {
				errRet(op, &reply.Status, NFS3ERR_STALE, inodes)
				done = true
				break
			}
			dipfrom = inodes[0]
			dipto = inodes[1]
		}

		util.DPrintf(5, "from %v to %v\n", dipfrom, dipto)

		frominumLookup, _ := dipfrom.lookupName(op, args.From.Name)
		frominum = frominumLookup
		if frominum == NULLINUM {
			errRet(op, &reply.Status, NFS3ERR_NOENT, inodes)
			done = true
			break
		}

		toInumLookup, _ := dipto.lookupName(op, args.To.Name)
		toinum = toInumLookup

		util.DPrintf(5, "frominum %d toinum %d\n", frominum, toinum)

		// rename to itself?
		if dipto == dipfrom && toinum == frominum {
			reply.Status = NFS3_OK
			commit(op, inodes)
			done = true
			break
		}

		// does to exist?
		if toinum != NULLINUM {
			// must lock 3 or 4 inodes in order
			var to *inode
			var from *inode
			abort(op, inodes)
			op = trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
			if dipto != dipfrom {
				inums := make([]fs.Inum, 4)
				inums[0] = dipfrom.inum
				inums[1] = dipto.inum
				inums[2] = frominum
				inums[3] = toinum
				inodes = lockInodes(op, inums)
				dipfrom = inodes[0]
				dipto = inodes[1]
				from = inodes[2]
				to = inodes[3]
			} else {
				inums := make([]fs.Inum, 3)
				inums[0] = dipfrom.inum
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
				if to.kind != from.kind {
					errRet(op, &reply.Status, NFS3ERR_INVAL, inodes)
					done = true
					break
				}
				if to.kind == NF3DIR && !to.isDirEmpty(op) {
					errRet(op, &reply.Status, NFS3ERR_NOTEMPTY, inodes)
					done = true
					break
				}
				ok := dipto.remName(op, args.To.Name)
				if !ok {
					errRet(op, &reply.Status, NFS3ERR_IO, inodes)
					done = true
					break
				}
				to.decLink(op)
				success = true
			} else { // retry
				abort(op, inodes)
			}
		} else {
			success = true
		}
	}
	if done {
		return reply
	}
	ok := dipfrom.remName(op, args.From.Name)
	if !ok {
		errRet(op, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	ok1 := dipto.addName(op, frominum, args.To.Name)
	if !ok1 {
		errRet(op, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	commitReply(op, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_LINK(args LINK3args) LINK3res {
	var reply LINK3res
	util.DPrintf(1, "NFS Link %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIR(args READDIR3args) READDIR3res {
	var reply READDIR3res
	util.DPrintf(1, "NFS Link %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args READDIRPLUS3args) READDIRPLUS3res {
	var reply READDIRPLUS3res
	util.DPrintf(1, "NFS ReadDirPlus %v\n", args)
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	ip := getInode(trans, args.Dir)
	if ip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inodes := []*inode{ip}
	if ip.kind != NF3DIR {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, inodes)
		return reply
	}
	dirlist := ip.ls3(trans, args.Cookie, args.Dircount)
	reply.Resok.Reply = dirlist
	commitReply(trans, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args FSSTAT3args) FSSTAT3res {
	var reply FSSTAT3res
	util.DPrintf(1, "NFS FsStat %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSINFO(args FSINFO3args) FSINFO3res {
	var reply FSINFO3res
	util.DPrintf(1, "NFS FsInfo %v\n", args)
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	reply.Resok.Wtmax = Uint32(trans.LogSzBytes())
	reply.Resok.Maxfilesize = Size3(maxFileSize())
	commitReply(trans, &reply.Status, []*inode{})
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args PATHCONF3args) PATHCONF3res {
	var reply PATHCONF3res
	util.DPrintf(1, "NFS PathConf %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

// RFC: forces or flushes data to stable storage that was previously
// written with a WRITE procedure call with the stable field set to
// UNSTABLE.
func (nfs *Nfs) NFSPROC3_COMMIT(args COMMIT3args) COMMIT3res {
	var reply COMMIT3res
	util.DPrintf(1, "NFS Commit %v\n", args)
	trans := trans.Begin(nfs.fs, nfs.txn, nfs.balloc, nfs.ialloc)
	ip := getInode(trans, args.File)
	fh := args.File.makeFh()
	if ip == nil {
		errRet(trans, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	if uint64(args.Offset)+uint64(args.Count) > ip.size {
		errRet(trans, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	ok := commitFh(trans, fh, []*inode{ip})
	if ok {
		reply.Status = NFS3_OK
	} else {
		errRet(trans, &reply.Status, NFS3ERR_IO, []*inode{ip})
	}
	return reply
}
