package goose_nfs

import ()

type Nfs struct {
	log *Log
	ic  *inodeCache
	fs  *FsSuper
}

func MkNfs() *Nfs {
	fs := mkFsSuper() // run first so that disk is initialized before mkLog
	log := mkLog()
	root := mkRootInode()
	rootblk := root.encode()
	fs.putRootBlk(ROOTINUM, rootblk)
	ic := mkInodeCache()
	return &Nfs{log: log, ic: ic, fs: fs}
}

// Returns locked inode on success
func (nfs *Nfs) getInode(tx *Txn, fh3 Nfs_fh3) *Inode {
	fh := fh3.makeFh()
	ip := nfs.ic.getInode(fh.ino)
	if ip == nil {
		return nil
	}
	ip.lock()
	if !ip.valid {
		ok := ip.load(tx, nfs.fs)
		if !ok {
			ip.unlock()
			return nil
		}
	}
	if ip.gen != fh.gen {
		ip.unlock()
		nfs.ic.putInode(ip)
		return nil
	}
	return ip
}

func (nfs *Nfs) GetAttr(args *GETATTR3args, reply *GETATTR3res) error {
	tx := Begin(nfs.log)
	ip := nfs.getInode(tx, args.Object)
	if ip == nil {
		reply.Status = NFS3ERR_STALE
		tx.Abort()
	} else {
		reply.Status = NFS3_OK
		reply.Resok.Obj_attributes = ip.mkFattr()
		tx.Commit()
		ip.unlock()
		nfs.ic.putInode(ip)
	}
	return nil
}

func (nfs *Nfs) Create(args *CREATE3args, reply *CREATE3res) error {
	tx := Begin(nfs.log)
	dip := nfs.getInode(tx, args.Where.Dir)
	if dip == nil {
		reply.Status = NFS3ERR_STALE
		tx.Abort()
		return nil
	}
	ip := allocInode(tx, NF3REG)
	if ip == nil {
		reply.Status = NFS3ERR_NOSPC
		tx.Abort()
		return nil
	}
	return nil
}
