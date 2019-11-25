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
	txn := Begin(nfs.log, nfs.bc)
	ip := nfs.getInode(txn, args.Object)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Obj_attributes = ip.mkFattr()
		txn.Commit()
		ip.unlockPut(nfs.ic, txn)
	}
	return nil
}

func (nfs *Nfs) SetAttr(args *SETATTR3args, reply *SETATTR3res) error {
	log.Printf("SetAttr %v\n", args)
	txn := Begin(nfs.log, nfs.bc)
	ip := nfs.getInode(txn, args.Object)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
	} else {
		if args.New_attributes.Size.Set_it {
			ok := ip.resize(nfs.fs, txn,
				uint64(args.New_attributes.Size.Size))
			if !ok {
				reply.Status = NFS3ERR_IO
				txn.Abort()
			} else {
				reply.Status = NFS3_OK
				txn.Commit()
			}
			ip.unlockPut(nfs.ic, txn)
		}
	}
	return nil
}

func (nfs *Nfs) Lookup(args *LOOKUP3args, reply *LOOKUP3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Lookup %v\n", args)
	dip := nfs.getInode(txn, args.What.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	inum := dip.lookupLink(txn, args.What.Name)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOENT
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ip := nfs.loadInode(txn, inum)
	if ip == nil {
		reply.Status = NFS3ERR_IO
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil

	}
	ip.lock()
	fh := Fh{ino: inum, gen: ip.gen}
	reply.Status = NFS3_OK
	reply.Resok.Object = fh.makeFh3()
	txn.Commit()
	dip.unlockPut(nfs.ic, txn)
	ip.unlockPut(nfs.ic, txn)
	return nil
}

func (nfs *Nfs) Read(args *READ3args, reply *READ3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Read %v\n", args.File)
	ip := nfs.getInode(txn, args.File)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		txn.Abort()
		ip.unlockPut(nfs.ic, txn)
		return nil
	}
	data, ok := ip.read(txn, uint64(args.Offset), uint64(args.Count))
	if !ok {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort()
		return nil
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(len(data))
		reply.Resok.Data = data
		txn.Commit()
	}
	ip.unlockPut(nfs.ic, txn)
	return nil
}

func (nfs *Nfs) Write(args *WRITE3args, reply *WRITE3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Write %v\n", args.File)
	ip := nfs.getInode(txn, args.File)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		txn.Abort()
		ip.unlockPut(nfs.ic, txn)
		return nil
	}
	count, ok := ip.write(txn, uint64(args.Offset), uint64(args.Count), args.Data)
	if !ok {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort()
		return nil
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Count = Count3(count)
		txn.Commit()
	}
	ip.unlockPut(nfs.ic, txn)
	return nil
}

// XXX deal with how
func (nfs *Nfs) Create(args *CREATE3args, reply *CREATE3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Create %v\n", args)
	dip := nfs.getInode(txn, args.Where.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	inum1 := dip.lookupLink(txn, args.Where.Name)
	if inum1 != NULLINUM {
		reply.Status = NFS3ERR_EXIST
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	inum := nfs.fs.allocInode(txn, NF3REG)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOSPC
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ok := dip.addLink(nfs.fs, txn, inum, args.Where.Name)
	if !ok {
		nfs.fs.freeInode(txn, inum)
		reply.Status = NFS3ERR_IO
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	txn.Commit()
	dip.unlockPut(nfs.ic, txn)
	return nil
}

func (nfs *Nfs) Remove(args *REMOVE3args, reply *REMOVE3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Remove %v\n", args)
	dip := nfs.getInode(txn, args.Object.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	inum := dip.lookupLink(txn, args.Object.Name)
	if inum == NULLINUM {
		reply.Status = NFS3ERR_NOENT
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ip := nfs.loadInode(txn, inum)
	if ip == nil {
		reply.Status = NFS3ERR_IO
		dip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ip.lock()
	if ip.kind != NF3REG {
		reply.Status = NFS3ERR_INVAL
		dip.unlockPut(nfs.ic, txn)
		ip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	n := dip.remlink(txn, args.Object.Name)
	if n == NULLINUM {
		reply.Status = NFS3ERR_IO
		dip.unlockPut(nfs.ic, txn)
		ip.unlockPut(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ip.unlink(nfs.fs, txn)
	txn.Commit()
	dip.unlockPut(nfs.ic, txn)
	ip.unlockPut(nfs.ic, txn)
	return nil
}
