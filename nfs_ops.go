package goose_nfs

import (
	"log"
)

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
	root := mkRootInode()
	rootblk := root.encode()
	fs.putRootBlk(ROOTINUM, rootblk)
	fs.markAlloc(fs.inodeStart() + fs.nInode)
	ic := mkCache()
	bc := mkCache()
	go l.Logger()
	return &Nfs{log: l, ic: ic, bc: bc, fs: fs}
}

func (nfs *Nfs) ShutdownNfs() {
	nfs.log.Shutdown()
}

// Returns locked inode on success
func (nfs *Nfs) getInode(txn *Txn, fh3 Nfs_fh3) *Inode {
	fh := fh3.makeFh()
	co := nfs.ic.getputObj(fh.ino)
	ip := nfs.fs.loadInode(txn, co, fh.ino)
	if ip == nil {
		log.Printf("loadInode failed\n")
		return nil
	}
	ip.lock()
	if ip.gen != fh.gen {
		log.Printf("wrong gen\n")
		ip.unlock()
		ip.putInode(nfs.ic, txn)
		return nil
	}
	return ip
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
		ip.unlock()
		ip.putInode(nfs.ic, txn)
	}
	return nil
}

func (nfs *Nfs) Create(args *CREATE3args, reply *CREATE3res) error {
	txn := Begin(nfs.log, nfs.bc)
	log.Printf("Create %v\n", args)
	dip := nfs.getInode(txn, args.Where.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	log.Printf("getInode %v\n", dip)
	inum := nfs.fs.allocInode(txn, NF3REG)
	if inum == 0 {
		reply.Status = NFS3ERR_NOSPC
		dip.unlock()
		dip.putInode(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	inum1 := dip.lookupLink(txn, args.Where.Name)
	if inum1 != 0 {
		reply.Status = NFS3ERR_EXIST
		dip.unlock()
		dip.putInode(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	ok := dip.addLink(nfs.fs, txn, inum, args.Where.Name)
	if !ok {
		reply.Status = NFS3ERR_IO
		dip.unlock()
		dip.putInode(nfs.ic, txn)
		txn.Abort()
		return nil
	}
	txn.Commit()
	dip.unlock()
	dip.putInode(nfs.ic, txn)
	return nil
}
