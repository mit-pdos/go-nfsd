package goose_nfs

import (
	"log"
)

const ICACHESZ uint64 = 10
const BCACHESZ uint64 = 10

type Nfs struct {
	log *Log
	ic  *Cache
	fs  *FsSuper
	bc  *Cache
}

func MkNfs() *Nfs {
	log.Printf("\nMake FsSuper\n")
	fs := mkFsSuper() // run first so that disk is initialized before mkLog
	l := mkLog(fs.nLog)
	if l == nil {
		panic("mkLog failed")
	}
	fs.initFs()
	ic := mkCache(ICACHESZ)
	bc := mkCache(BCACHESZ)
	go l.Logger()
	return &Nfs{log: l, ic: ic, bc: bc, fs: fs}
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.log.Shutdown()
}

func (nfs *Nfs) GetAttr(args *GETATTR3args, reply *GETATTR3res) error {
	log.Printf("GetAttr %v\n", args)
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	ip := getInode(txn, args.Object)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Obj_attributes = ip.mkFattr()
		txn.Commit([]*Inode{ip})
	}
	return nil
}

func (nfs *Nfs) SetAttr(args *SETATTR3args, reply *SETATTR3res) error {
	log.Printf("SetAttr %v\n", args)
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	ip := getInode(txn, args.Object)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
	} else {
		if args.New_attributes.Size.Set_it {
			ok := ip.resize(txn,
				uint64(args.New_attributes.Size.Size))
			if !ok {
				reply.Status = NFS3ERR_IO
				txn.Abort([]*Inode{ip})
			} else {
				reply.Status = NFS3_OK
				txn.Commit([]*Inode{ip})
			}
		} else {
			reply.Status = NFS3ERR_NOTSUPP
			txn.Abort([]*Inode{ip})
		}
	}
	return nil
}

func (nfs *Nfs) Lookup(args *LOOKUP3args, reply *LOOKUP3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Lookup %v\n", args)
	dip := getInode(txn, args.What.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
		return nil
	}
	inum := dip.lookupLink(txn, args.What.Name)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOENT
		txn.Abort([]*Inode{dip})
		return nil
	}
	ip := loadInode(txn, inum)
	if ip == nil {
		reply.Status = NFS3ERR_IO
		txn.Abort([]*Inode{dip})
		return nil

	}
	ip.lock()
	fh := Fh{ino: inum, gen: ip.gen}
	reply.Status = NFS3_OK
	reply.Resok.Object = fh.makeFh3()
	txn.Commit([]*Inode{ip, dip})
	return nil
}

func (nfs *Nfs) Access(args *ACCESS3args, reply *ACCESS3res) error {
	log.Printf("Access %v\n", args)
	reply.Status = NFS3_OK
	reply.Resok.Access = ACCESS3_READ | ACCESS3_LOOKUP | ACCESS3_MODIFY | ACCESS3_EXTEND | ACCESS3_DELETE | ACCESS3_EXECUTE
	return nil
}

func (nfs *Nfs) ReadLink(args *READLINK3args, reply *READLINK3res) error {
	log.Printf("ReadLink %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) Read(args *READ3args, reply *READ3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Read %v\n", args.File)
	ip := getInode(txn, args.File)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
		return nil
	}
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		txn.Abort([]*Inode{ip})
		return nil
	}
	data, ok := ip.read(txn, uint64(args.Offset), uint64(args.Count))
	if !ok {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort([]*Inode{ip})
		return nil
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(len(data))
		reply.Resok.Data = data
		txn.Commit([]*Inode{ip})
	}
	return nil
}

// XXX deal with stable_how and committed
func (nfs *Nfs) Write(args *WRITE3args, reply *WRITE3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Write %v\n", args.File)
	ip := getInode(txn, args.File)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
		return nil
	}
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		txn.Abort([]*Inode{ip})
		return nil
	}
	count, ok := ip.write(txn, uint64(args.Offset), uint64(args.Count), args.Data)
	if !ok {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort([]*Inode{ip})
		return nil
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Committed = FILE_SYNC
		reply.Resok.Count = Count3(count)
		txn.Commit([]*Inode{ip})
	}
	return nil
}

// XXX deal with how
// XXX Check for . and ..
func (nfs *Nfs) Create(args *CREATE3args, reply *CREATE3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Create %v\n", args)
	dip := getInode(txn, args.Where.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
		return nil
	}
	inum1 := dip.lookupLink(txn, args.Where.Name)
	if inum1 != NULLINUM {
		reply.Status = NFS3ERR_EXIST
		txn.Abort([]*Inode{dip})
		return nil
	}
	inum := nfs.fs.allocInode(txn, NF3REG)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort([]*Inode{dip})
		return nil
	}
	ok := dip.addLink(txn, inum, args.Where.Name)
	if !ok {
		nfs.fs.freeInum(txn, inum)
		reply.Status = NFS3ERR_IO
		txn.Abort([]*Inode{dip})
		return nil
	}
	txn.Commit([]*Inode{dip})
	reply.Status = NFS3_OK
	return nil
}

func (nfs *Nfs) MakeDir(args *MKDIR3args, reply *MKDIR3res) error {
	log.Printf("MakeDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) SymLink(args *SYMLINK3args, reply *SYMLINK3res) error {
	log.Printf("MakeDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) MakeNod(args *MKNOD3args, reply *MKNOD3res) error {
	log.Printf("MakeDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

// XXX Check for . and ..
func (nfs *Nfs) Remove(args *REMOVE3args, reply *REMOVE3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Remove %v\n", args)
	dip := getInode(txn, args.Object.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort(nil)
		return nil
	}
	inum := dip.lookupLink(txn, args.Object.Name)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOENT
		txn.Abort([]*Inode{dip})
		return nil
	}
	ip := loadInode(txn, inum)
	if ip == nil {
		reply.Status = NFS3ERR_IO
		txn.Abort([]*Inode{ip})
		return nil
	}
	ip.lock()
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		txn.Abort([]*Inode{ip, dip})
		return nil
	}
	n := dip.remLink(txn, args.Object.Name)
	if n == NULLINUM {
		reply.Status = NFS3ERR_IO
		txn.Abort([]*Inode{ip, dip})
		return nil
	}
	ip.decLink(txn)
	txn.Commit([]*Inode{ip, dip})
	reply.Status = NFS3_OK
	return nil
}

func (nfs *Nfs) RmDir(args *RMDIR3args, reply *RMDIR3res) error {
	log.Printf("RmDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

// XXX check for . and ..
func (nfs *Nfs) Rename(args *RENAME3args, reply *RENAME3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Rename %v\n", args)

	toh := args.To.Dir.makeFh()
	fromh := args.From.Dir.makeFh()

	inodes := make([]*Inode, 0, 4)
	var dipto *Inode
	var dipfrom *Inode

	if args.From.Dir.equal(args.To.Dir) {
		dipfrom = getInode(txn, args.From.Dir)
		if dipfrom == nil {
			reply.Status = NFS3ERR_STALE
			txn.Abort(nil)
			return nil
		}
		dipto = dipfrom
		inodes = append(inodes, dipfrom)
	} else {
		var ok bool
		if toh.ino < fromh.ino {
			inodes, ok = getInodes(txn, []Fh{toh, fromh})
		} else {
			inodes, ok = getInodes(txn, []Fh{fromh, toh})
		}
		if !ok {
			reply.Status = NFS3ERR_STALE
			txn.Abort(inodes)
		}
		if toh.ino < fromh.ino {
			dipto = inodes[0]
			dipfrom = inodes[1]
		} else {
			dipto = inodes[1]
			dipfrom = inodes[0]
		}
	}

	log.Printf("from %v to %v\n", dipfrom, dipto)

	frominum := dipfrom.lookupLink(txn, args.From.Name)
	if frominum == NULLINUM {
		reply.Status = NFS3ERR_NOENT
		txn.Abort(inodes)
		return nil
	}

	toinum := dipto.lookupLink(txn, args.To.Name)

	log.Printf("frominum %d toinum %d\n", frominum, toinum)

	// rename to itself?
	if dipto == dipfrom && toinum == frominum {
		reply.Status = NFS3_OK
		txn.Commit(inodes)
		return nil
	}

	// does to exist?
	if toinum != NULLINUM {
		to := getInodeInum(txn, toinum)
		if to == nil {
			reply.Status = NFS3ERR_IO
			txn.Abort(inodes)
			return nil
		}
		inodes = append(inodes, to)
		from := getInodeInum(txn, frominum)
		if from == nil {
			reply.Status = NFS3ERR_IO
			txn.Abort(inodes)
			return nil
		}
		inodes = append(inodes, from)
		if to.kind != from.kind {
			reply.Status = NFS3ERR_INVAL
			txn.Abort(inodes)
			return nil
		}
		if to.kind == NF3DIR && !to.dirEmpty(txn) {
			reply.Status = NFS3ERR_NOTEMPTY
			txn.Abort(inodes)
			return nil
		}
		n := dipto.remLink(txn, args.To.Name)
		if n == NULLINUM {
			reply.Status = NFS3ERR_IO
			txn.Abort(inodes)
			return nil
		}
		to.decLink(txn)
	}
	n := dipfrom.remLink(txn, args.From.Name)
	if n == NULLINUM {
		reply.Status = NFS3ERR_IO
		txn.Abort(inodes)
		return nil
	}
	ok := dipto.addLink(txn, frominum, args.To.Name)
	if !ok {
		reply.Status = NFS3ERR_IO
		txn.Abort(inodes)
		return nil
	}
	txn.Commit(inodes)
	reply.Status = NFS3_OK
	return nil
}

func (nfs *Nfs) Link(args *LINK3args, reply *LINK3res) error {
	log.Printf("Link %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) ReadDir(args *READDIR3args, reply *READDIR3res) error {
	log.Printf("ReadDir %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) ReadDirPlus(args *READDIRPLUS3args, reply *READDIRPLUS3res) error {
	log.Printf("ReadDirPlus %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) FsStat(args *FSSTAT3args, reply *FSSTAT3res) error {
	log.Printf("FsStat %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) FsInfo(args *FSINFO3args, reply *FSINFO3res) error {
	log.Printf("FsInfo %v\n", args)
	reply.Status = NFS3_OK
	reply.Resok.Maxfilesize = Size3(NDIRECT)
	return nil
}

func (nfs *Nfs) PathConf(args *PATHCONF3args, reply *PATHCONF3res) error {
	log.Printf("PathConf %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

func (nfs *Nfs) COMMIT(args *COMMIT3args, reply *COMMIT3res) error {
	log.Printf("Commit %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}
