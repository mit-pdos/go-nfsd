package goose_nfs

import (
	"github.com/zeldovich/go-rpcgen/xdr"
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

func errRet(txn *Txn, status *Nfsstat3, err Nfsstat3, inodes []*Inode) error {
	*status = err
	txn.Abort(inodes)
	return nil
}

func (nfs *Nfs) NullNFS(args *xdr.Void, reply *xdr.Void) error {
	log.Printf("Null\n")
	return nil
}

func (nfs *Nfs) GetAttr(args *GETATTR3args, reply *GETATTR3res) error {
	log.Printf("GetAttr %v\n", args)
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	ip := getInode(txn, args.Object)
	if ip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
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
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	} else {
		if args.New_attributes.Size.Set_it {
			ok := ip.resize(txn,
				uint64(args.New_attributes.Size.Size))
			if !ok {
				return errRet(txn, &reply.Status, NFS3ERR_IO, []*Inode{ip})
			} else {
				reply.Status = NFS3_OK
				txn.Commit([]*Inode{ip})
			}
		} else {
			return errRet(txn, &reply.Status, NFS3ERR_NOTSUPP, []*Inode{ip})
		}
	}
	return nil
}

func (nfs *Nfs) Lookup(args *LOOKUP3args, reply *LOOKUP3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Lookup %v\n", args)
	dip := getInode(txn, args.What.Dir)
	if dip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	}
	inum := dip.lookupLink(txn, args.What.Name)
	if inum == NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_NOENT, []*Inode{dip})
	}
	ip := loadInode(txn, inum)
	if ip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_IO, []*Inode{dip})

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
	reply.Resok.Access = Uint32(ACCESS3_READ | ACCESS3_LOOKUP | ACCESS3_MODIFY | ACCESS3_EXTEND | ACCESS3_DELETE | ACCESS3_EXECUTE)
	return nil
}

func (nfs *Nfs) ReadLink(args *READLINK3args, reply *READLINK3res) error {
	log.Printf("ReadLink %v\n", args)
	reply.Status = NFS3ERR_NOTSUPP
	return nil
}

// XXX eof
func (nfs *Nfs) Read(args *READ3args, reply *READ3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Read %v\n", args.File)
	ip := getInode(txn, args.File)
	if ip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	}
	if ip.kind != NF3REG {
		return errRet(txn, &reply.Status, NFS3ERR_INVAL, []*Inode{ip})
	}
	data, eof, ok := ip.read(txn, uint64(args.Offset), uint64(args.Count))
	if !ok {
		return errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*Inode{ip})
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(len(data))
		reply.Resok.Data = data
		reply.Resok.Eof = eof
		txn.Commit([]*Inode{ip})
	}
	return nil
}

// XXX deal with stable_how and committed
// XXX Mtime
func (nfs *Nfs) Write(args *WRITE3args, reply *WRITE3res) error {
	txn := Begin(nfs.log, nfs.bc, nfs.fs, nfs.ic)
	log.Printf("Write %v off %d cnt %d\n", args.File, args.Offset, args.Count)
	ip := getInode(txn, args.File)
	fh := args.File.makeFh()
	if ip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	}
	if ip.kind != NF3REG {
		return errRet(txn, &reply.Status, NFS3ERR_INVAL, []*Inode{ip})
	}
	count, ok := ip.write(txn, uint64(args.Offset), uint64(args.Count), args.Data)
	if !ok {
		return errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*Inode{ip})
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(count)
		how, _ := txn.CommitHow([]*Inode{ip}, fh, args.Stable)
		reply.Resok.Committed = how
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
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	}
	inum1 := dip.lookupLink(txn, args.Where.Name)
	if inum1 != NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_EXIST, []*Inode{dip})
	}
	inum := nfs.fs.allocInode(txn, NF3REG)
	if inum == NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_NOSPC, []*Inode{dip})
	}
	ok := dip.addLink(txn, inum, args.Where.Name)
	if !ok {
		nfs.fs.freeInum(txn, inum)
		return errRet(txn, &reply.Status, NFS3ERR_IO, []*Inode{dip})
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
		return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
	}
	inum := dip.lookupLink(txn, args.Object.Name)
	if inum == NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_NOENT, []*Inode{dip})
	}
	ip := loadInode(txn, inum)
	if ip == nil {
		return errRet(txn, &reply.Status, NFS3ERR_IO, []*Inode{ip, dip})
	}
	ip.lock()
	if ip.kind != NF3REG {
		return errRet(txn, &reply.Status, NFS3ERR_INVAL, []*Inode{ip, dip})
	}
	n := dip.remLink(txn, args.Object.Name)
	if n == NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_IO, []*Inode{ip, dip})
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
			return errRet(txn, &reply.Status, NFS3ERR_STALE, nil)
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
			return errRet(txn, &reply.Status, NFS3ERR_STALE, inodes)
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
		return errRet(txn, &reply.Status, NFS3ERR_NOENT, inodes)
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
			return errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		}
		inodes = append(inodes, to)
		from := getInodeInum(txn, frominum)
		if from == nil {
			return errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		}
		inodes = append(inodes, from)
		if to.kind != from.kind {
			return errRet(txn, &reply.Status, NFS3ERR_INVAL, inodes)
		}
		if to.kind == NF3DIR && !to.dirEmpty(txn) {
			return errRet(txn, &reply.Status, NFS3ERR_NOTEMPTY, inodes)
		}
		n := dipto.remLink(txn, args.To.Name)
		if n == NULLINUM {
			return errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
		}
		to.decLink(txn)
	}
	n := dipfrom.remLink(txn, args.From.Name)
	if n == NULLINUM {
		return errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
	}
	ok := dipto.addLink(txn, frominum, args.To.Name)
	if !ok {
		return errRet(txn, &reply.Status, NFS3ERR_IO, inodes)
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
