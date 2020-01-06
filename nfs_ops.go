package goose_nfs

import (
	"sort"
	"sync"

	"github.com/tchajed/goose/machine"
)

const ICACHESZ uint64 = 20            // XXX resurrect icache
const BCACHESZ uint64 = HDRADDRS + 10 // At least as big as log

type Nfs struct {
	mu       *sync.Mutex
	condShut *sync.Cond
	log      *walog
	fs       *fsSuper
	bc       *cache
	balloc   *alloc
	ialloc   *alloc
	locks    *lockMap
	// locked   *addrMap

	commit  *commit
	nthread uint32
}

// XXX call recovery, once nfs uses persistent storage
func MkNfs() *Nfs {
	fs := mkFsSuper() // run first so that disk is initialized before mkLog
	dPrintf(1, "Super: %v\n", fs)

	l := mkLog()
	if l == nil {
		panic("mkLog failed")
	}

	fs.initFs()
	bc := mkCache(BCACHESZ)

	// TODO: do we still need to use machine.Spawn,
	//  or can we just use go statements?
	machine.Spawn(func() { l.logger() })
	machine.Spawn(func() { l.installer() })

	mu := new(sync.Mutex)
	nfs := &Nfs{
		mu:       mu,
		condShut: sync.NewCond(mu),
		log:      l,
		fs:       fs,
		bc:       bc,
		balloc:   mkAlloc(fs.bitmapBlockStart(), fs.nBlockBitmap, BBMAP),
		ialloc:   mkAlloc(fs.bitmapInodeStart(), fs.nInodeBitmap, IBMAP),
		locks:    mkLockMap(),
		commit:   mkcommit(),
		nthread:  0,
	}
	nfs.makeRootDir()
	return nfs
}

func (nfs *Nfs) makeRootDir() {
	txn := begin(nfs)
	ip := getInodeInum(txn, ROOTINUM)
	if ip == nil {
		panic("makeRootDir")
	}
	ip.mkRootDir(txn)
	ok := txn.commit([]*inode{ip})
	if !ok {
		panic("makeRootDir")
	}
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.mu.Lock()
	for nfs.nthread > 0 {
		dPrintf(1, "ShutdownNfs: wait %d\n", nfs.nthread)
		nfs.condShut.Wait()
	}
	nfs.mu.Unlock()
	nfs.log.doShutdown()
}

func errRet(txn *txn, status *Nfsstat3, err Nfsstat3, inodes []*inode) {
	*status = err
	txn.abort(inodes)
}

func commitReply(txn *txn, status *Nfsstat3, inodes []*inode) {
	ok := txn.commit(inodes)
	if ok {
		*status = NFS3_OK
	} else {
		*status = NFS3ERR_SERVERFAULT
	}
}

func (nfs *Nfs) NFSPROC3_NULL() {
	dPrintf(1, "NFS Null\n")
}

// XXX factor out lookup ip, test, and maybe fail pattern
func (nfs *Nfs) NFSPROC3_GETATTR(args GETATTR3args) GETATTR3res {
	var reply GETATTR3res
	dPrintf(1, "NFS GetAttr %v\n", args)
	txn := begin(nfs)
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
	dPrintf(1, "NFS SetAttr %v\n", args)
	txn := begin(nfs)
	ip := getInode(txn, args.Object)
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if args.New_attributes.Size.Set_it {
		ip.resize(txn, uint64(args.New_attributes.Size.Size))
		commitReply(txn, &reply.Status, []*inode{ip})
	} else {
		errRet(txn, &reply.Status, NFS3ERR_NOTSUPP, []*inode{ip})
	}
	return reply
}

// Lock inodes in sorted order, but return the pointers in the same order as in inums
// Caller must revalidate inodes.
func lockInodes(txn *txn, inums []inum) []*inode {
	dPrintf(1, "lock inodes %v\n", inums)
	sorted := make([]inum, len(inums))
	copy(sorted, inums)
	sort.Slice(sorted, func(i, j int) bool { return inums[i] < inums[j] })
	var inodes = make([]*inode, len(inums))
	for _, inm := range sorted {
		ip := getInodeInum(txn, inm)
		if ip == nil {
			txn.abort(inodes)
			return nil
		}
		// put in same position as in inums
		pos := func(inm inum) int {
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

func twoInums(inum1, inum2 inum) []inum {
	inums := make([]inum, 2)
	inums[0] = inum1
	inums[1] = inum2
	return inums
}

// First lookup inode up for child, then for parent, because parent
// inum > child inum and then revalidate that child is still in parent
// directory.
func (nfs *Nfs) lookupOrdered(txn *txn, name Filename3, parent fh, inm inum) []*inode {
	dPrintf(5, "NFS lookupOrdered child %d parent %v\n", inm, parent)
	inodes := lockInodes(txn, twoInums(inm, parent.ino))
	if inodes == nil {
		return nil
	}
	dip := inodes[1]
	if dip.gen != parent.gen {
		txn.abort(inodes)
		return nil
	}
	child, _ := dip.lookupName(txn, name)
	if child == NULLINUM || child != inm {
		txn.abort(inodes)
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
	var txn *txn
	var done bool = false
	for ip == nil {
		txn = begin(nfs)
		dPrintf(1, "NFS Lookup %v\n", args)
		dip := getInode(txn, args.What.Dir)
		if dip == nil {
			errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
			done = true
			break
		}
		inum, _ := dip.lookupName(txn, args.What.Name)
		if inum == NULLINUM {
			errRet(txn, &reply.Status, NFS3ERR_NOENT, []*inode{dip})
			done = true
			break
		}
		inodes = []*inode{dip}
		if inum == dip.inum {
			ip = dip
		} else {
			if inum < dip.inum {
				// Abort. Try to lock inodes in order
				txn.abort([]*inode{dip})
				parent := args.What.Dir.makeFh()
				txn = begin(nfs)
				inodes = nfs.lookupOrdered(txn, args.What.Name,
					parent, inum)
				if inodes == nil {
					ip = nil
				} else {
					ip = inodes[0]
				}
			} else {
				ip = getInodeLocked(txn, inum)
				inodes = twoInodes(ip, dip)
			}
		}
	}
	if done {
		return reply
	}
	fh := fh{ino: ip.inum, gen: ip.gen}
	reply.Resok.Object = fh.makeFh3()
	commitReply(txn, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_ACCESS(args ACCESS3args) ACCESS3res {
	var reply ACCESS3res
	dPrintf(1, "NFS Access %v\n", args)
	reply.Status = NFS3_OK
	reply.Resok.Access = Uint32(ACCESS3_READ | ACCESS3_LOOKUP | ACCESS3_MODIFY | ACCESS3_EXTEND | ACCESS3_DELETE | ACCESS3_EXECUTE)
	return reply
}

func (nfs *Nfs) NFSPROC3_READLINK(args READLINK3args) READLINK3res {
	var reply READLINK3res
	dPrintf(1, "NFS ReadLink %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READ(args READ3args) READ3res {
	var reply READ3res
	txn := begin(nfs)
	dPrintf(1, "NFS Read %v %d %d\n", args.File, args.Offset, args.Count)
	ip := getInode(txn, args.File)
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	data, eof := ip.read(txn, uint64(args.Offset), uint64(args.Count))
	reply.Resok.Count = Count3(len(data))
	reply.Resok.Data = data
	reply.Resok.Eof = eof
	commitReply(txn, &reply.Status, []*inode{ip})
	return reply
}

// XXX Mtime
func (nfs *Nfs) NFSPROC3_WRITE(args WRITE3args) WRITE3res {
	var reply WRITE3res
	txn := begin(nfs)
	dPrintf(1, "NFS Write %v off %d cnt %d how %d\n", args.File, args.Offset,
		args.Count, args.Stable)
	ip := getInode(txn, args.File)
	fh := args.File.makeFh()
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	if uint64(args.Count) >= maxLogSize() {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	count, writeOk := ip.write(txn, uint64(args.Offset), uint64(args.Count),
		args.Data)
	if !writeOk {
		errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*inode{ip})
		return reply
	}
	var ok = true
	if args.Stable == FILE_SYNC {
		// RFC: "FILE_SYNC, the server must commit the
		// data written plus all file system metadata
		// to stable storage before returning results."
		ok = txn.commit([]*inode{ip})
	} else if args.Stable == DATA_SYNC {
		// RFC: "DATA_SYNC, then the server must commit
		// all of the data to stable storage and
		// enough of the metadata to retrieve the data
		// before returning."
		ok = txn.commitData([]*inode{ip}, fh)
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
		ok = txn.commitUnstable([]*inode{ip}, fh)
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
	txn := begin(nfs)
	dPrintf(1, "NFS Create %v\n", args)
	dip := getInode(txn, args.Where.Dir)
	if dip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dip.lookupName(txn, args.Where.Name)
	if inum1 != NULLINUM {
		errRet(txn, &reply.Status, NFS3ERR_EXIST, []*inode{dip})
		return reply
	}
	inum := allocInode(txn, NF3REG)
	if inum == NULLINUM {
		errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*inode{dip})
		return reply
	}
	ok := dip.addName(txn, inum, args.Where.Name)
	if !ok {
		freeInum(txn, inum)
		errRet(txn, &reply.Status, NFS3ERR_IO, []*inode{dip})
		return reply
	}
	commitReply(txn, &reply.Status, []*inode{dip})
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
	txn := begin(nfs)
	dPrintf(1, "NFS MakeDir %v\n", args)
	dip := getInode(txn, args.Where.Dir)
	if dip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inum1, _ := dip.lookupName(txn, args.Where.Name)
	if inum1 != NULLINUM {
		errRet(txn, &reply.Status, NFS3ERR_EXIST, []*inode{dip})
		return reply
	}
	inum := allocInode(txn, NF3DIR)
	if inum == NULLINUM {
		errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*inode{dip})
		return reply
	}
	ip := getInodeLocked(txn, inum)
	ok := ip.initDir(txn, dip.inum)
	if !ok {
		ip.decLink(txn)
		errRet(txn, &reply.Status, NFS3ERR_NOSPC, twoInodes(dip, ip))
		return reply
	}
	ok1 := dip.addName(txn, inum, args.Where.Name)
	if !ok1 {
		ip.decLink(txn)
		errRet(txn, &reply.Status, NFS3ERR_IO, twoInodes(dip, ip))
		return reply
	}
	dip.nlink = dip.nlink + 1 // for ..
	dip.writeInode(txn)
	commitReply(txn, &reply.Status, twoInodes(dip, ip))
	return reply
}

func (nfs *Nfs) NFSPROC3_SYMLINK(args SYMLINK3args) SYMLINK3res {
	var reply SYMLINK3res
	dPrintf(1, "NFS MakeDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_MKNOD(args MKNOD3args) MKNOD3res {
	var reply MKNOD3res
	dPrintf(1, "NFS MakeNod %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_REMOVE(args REMOVE3args) REMOVE3res {
	var reply REMOVE3res
	var ip *inode
	var dip *inode
	var inodes []*inode
	var txn *txn
	var done bool = false
	for ip == nil {
		txn = begin(nfs)
		dPrintf(1, "NFS Remove %v\n", args)
		dip = getInode(txn, args.Object.Dir)
		if dip == nil {
			errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
			done = true
			break
		}
		if illegalName(args.Object.Name) {
			errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{dip})
			done = true
			break
		}
		inum, _ := dip.lookupName(txn, args.Object.Name)
		if inum == NULLINUM {
			errRet(txn, &reply.Status, NFS3ERR_NOENT, []*inode{dip})
			done = true
			break
		}
		if inum < dip.inum {
			// Abort. Try to lock inodes in order
			txn.abort([]*inode{dip})
			txn := begin(nfs)
			parent := args.Object.Dir.makeFh()
			inodes = nfs.lookupOrdered(txn, args.Object.Name, parent, inum)
			ip = inodes[0]
			dip = inodes[1]
		} else {
			ip = getInodeLocked(txn, inum)
			inodes = twoInodes(ip, dip)
		}
	}
	if done {
		return reply
	}
	if ip.kind != NF3REG {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, inodes)
		return reply
	}
	ok := dip.remName(txn, args.Object.Name)
	if !ok {
		errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	ip.decLink(txn)
	commitReply(txn, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_RMDIR(args RMDIR3args) RMDIR3res {
	var reply RMDIR3res
	dPrintf(1, "NFS RmDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func validateRename(txn *txn, inodes []*inode, fromfh fh, tofh fh,
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
		dPrintf(10, "revalidate ino failed\n")
		return false
	}
	frominum, _ := dipfrom.lookupName(txn, fromn)
	toinum, _ := dipto.lookupName(txn, ton)
	if from.inum != frominum || toinum != to.inum {
		dPrintf(10, "revalidate inums failed\n")
		return false
	}
	return true
}

func (nfs *Nfs) NFSPROC3_RENAME(args RENAME3args) RENAME3res {
	var reply RENAME3res
	var dipto *inode
	var dipfrom *inode
	var txn *txn
	var inodes []*inode
	var frominum inum
	var toinum inum
	var success bool = false
	var done bool = false

	for !success {
		txn = begin(nfs)
		dPrintf(1, "NFS Rename %v\n", args)

		toh := args.To.Dir.makeFh()
		fromh := args.From.Dir.makeFh()

		if illegalName(args.From.Name) {
			errRet(txn, &reply.Status, NFS3ERR_INVAL, nil)
			done = true
			break
		}

		if args.From.Dir.equal(args.To.Dir) {
			dipfrom = getInode(txn, args.From.Dir)
			if dipfrom == nil {
				errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
				done = true
				break
			}
			dipto = dipfrom
			inodes = []*inode{dipfrom}
		} else {
			inodes = lockInodes(txn, twoInums(fromh.ino, toh.ino))
			if inodes == nil {
				errRet(txn, &reply.Status, NFS3ERR_STALE, inodes)
				done = true
				break
			}
			dipfrom = inodes[0]
			dipto = inodes[1]
		}

		dPrintf(5, "from %v to %v\n", dipfrom, dipto)

		frominumLookup, _ := dipfrom.lookupName(txn, args.From.Name)
		frominum = frominumLookup
		if frominum == NULLINUM {
			errRet(txn, &reply.Status, NFS3ERR_NOENT, inodes)
			done = true
			break
		}

		toInumLookup, _ := dipto.lookupName(txn, args.To.Name)
		toinum = toInumLookup

		dPrintf(5, "frominum %d toinum %d\n", frominum, toinum)

		// rename to itself?
		if dipto == dipfrom && toinum == frominum {
			reply.Status = NFS3_OK
			txn.commit(inodes)
			done = true
			break
		}

		// does to exist?
		if toinum != NULLINUM {
			// must lock 3 or 4 inodes in order
			var to *inode
			var from *inode
			txn.abort(inodes)
			txn = begin(nfs)
			if dipto != dipfrom {
				inums := make([]inum, 4)
				inums[0] = dipfrom.inum
				inums[1] = dipto.inum
				inums[2] = frominum
				inums[3] = toinum
				inodes = lockInodes(txn, inums)
				dipfrom = inodes[0]
				dipto = inodes[1]
				from = inodes[2]
				to = inodes[3]
			} else {
				inums := make([]inum, 3)
				inums[0] = dipfrom.inum
				inums[1] = frominum
				inums[2] = toinum
				inodes = lockInodes(txn, inums)
				dipfrom = inodes[0]
				dipto = inodes[0]
				from = inodes[1]
				to = inodes[2]
			}
			dPrintf(5, "inodes %v\n", inodes)
			if validateRename(txn, inodes, fromh, toh,
				args.From.Name, args.To.Name) {
				if to.kind != from.kind {
					errRet(txn, &reply.Status, NFS3ERR_INVAL, inodes)
					done = true
					break
				}
				if to.kind == NF3DIR && !to.isDirEmpty(txn) {
					errRet(txn, &reply.Status, NFS3ERR_NOTEMPTY, inodes)
					done = true
					break
				}
				ok := dipto.remName(txn, args.To.Name)
				if !ok {
					errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
					done = true
					break
				}
				to.decLink(txn)
				success = true
			} else { // retry
				txn.abort(inodes)
			}
		} else {
			success = true
		}
	}
	if done {
		return reply
	}
	ok := dipfrom.remName(txn, args.From.Name)
	if !ok {
		errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	ok1 := dipto.addName(txn, frominum, args.To.Name)
	if !ok1 {
		errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		return reply
	}
	commitReply(txn, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_LINK(args LINK3args) LINK3res {
	var reply LINK3res
	dPrintf(1, "NFS Link %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIR(args READDIR3args) READDIR3res {
	var reply READDIR3res
	dPrintf(1, "NFS Link %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_READDIRPLUS(args READDIRPLUS3args) READDIRPLUS3res {
	var reply READDIRPLUS3res
	dPrintf(1, "NFS ReadDirPlus %v\n", args)
	txn := begin(nfs)
	ip := getInode(txn, args.Dir)
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	inodes := []*inode{ip}
	if ip.kind != NF3DIR {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, inodes)
		return reply
	}
	dirlist := ip.ls3(txn, args.Cookie, args.Dircount)
	reply.Resok.Reply = dirlist
	commitReply(txn, &reply.Status, inodes)
	return reply
}

func (nfs *Nfs) NFSPROC3_FSSTAT(args FSSTAT3args) FSSTAT3res {
	var reply FSSTAT3res
	dPrintf(1, "NFS FsStat %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

func (nfs *Nfs) NFSPROC3_FSINFO(args FSINFO3args) FSINFO3res {
	var reply FSINFO3res
	dPrintf(1, "NFS FsInfo %v\n", args)
	reply.Status = NFS3_OK
	reply.Resok.Wtmax = Uint32(maxLogSize())
	reply.Resok.Maxfilesize = Size3(maxFileSize())
	// XXX maybe set wtpref, wtmult, and rdmult
	return reply
}

func (nfs *Nfs) NFSPROC3_PATHCONF(args PATHCONF3args) PATHCONF3res {
	var reply PATHCONF3res
	dPrintf(1, "NFS PathConf %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return reply
}

// RFC: forces or flushes data to stable storage that was previously
// written with a WRITE procedure call with the stable field set to
// UNSTABLE.
func (nfs *Nfs) NFSPROC3_COMMIT(args COMMIT3args) COMMIT3res {
	var reply COMMIT3res
	dPrintf(1, "NFS Commit %v\n", args)
	txn := begin(nfs)
	ip := getInode(txn, args.File)
	fh := args.File.makeFh()
	if ip == nil {
		errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
		return reply
	}
	if ip.kind != NF3REG {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	if uint64(args.Offset)+uint64(args.Count) > ip.size {
		errRet(txn, &reply.Status, NFS3ERR_INVAL, []*inode{ip})
		return reply
	}
	ok := txn.commitFh(fh, []*inode{ip})
	if ok {
		reply.Status = NFS3_OK
	} else {
		errRet(txn, &reply.Status, NFS3ERR_IO, []*inode{ip})
	}
	return reply
}
