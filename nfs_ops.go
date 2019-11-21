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
	fs := mkFsSuper() // run first so that disk is initialized before mkLog
	l := mkLog()
	root := mkRootInode()
	rootblk := root.encode()
	fs.putRootBlk(ROOTINUM, rootblk)
	ic := mkCache()
	bc := mkCache()
	go l.Logger()
	return &Nfs{log: l, ic: ic, bc: bc, fs: fs}
}

// Returns locked inode on success
func (nfs *Nfs) getInode(txn *Txn, fh3 Nfs_fh3) *Inode {
	fh := fh3.makeFh()
	co := nfs.ic.getputObj(fh.ino)
	ip := nfs.fs.loadInode(txn, co, fh.ino)
	if ip == nil {
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
		log.Println("putinode")
		ip.putInode(nfs.ic, txn)
	}
	return nil
}

func (nfs *Nfs) Create(args *CREATE3args, reply *CREATE3res) error {
	txn := Begin(nfs.log, nfs.bc)
	dip := nfs.getInode(txn, args.Where.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		txn.Abort()
		return nil
	}
	ip := allocInode(txn, NF3REG)
	if ip == nil {
		reply.Status = NFS3ERR_NOSPC
		txn.Abort()
		return nil
	}
	return nil
}
