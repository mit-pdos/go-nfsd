package barebones

import (
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/buftxn"
)

func (nfs *BarebonesNfs) OpGetAttr(fh Fh) (*inode.Inode, Nfsstat3) {
	buftxn := buftxn.Begin(nfs.txn)
	return nfs.getInodeByFh(buftxn, fh)
}

func (nfs *BarebonesNfs) OpLookup(dfh Fh, name string) (*inode.Inode, *inode.Inode, Nfsstat3) {
	buftxn := buftxn.Begin(nfs.txn)

	dip, err1 := nfs.getInodeByFh(buftxn, dfh)
	if err1 != NFS3_OK {
		return nil, nil, err1
	}
	in := nfs.lookupName(dip, name)
	if in == 0 {
		return nil, nil, NFS3ERR_NOENT
	}
	ip := nfs.getInode(buftxn, in)
	return dip, ip, NFS3_OK
}

func (nfs *BarebonesNfs) OpMkdir(dfh Fh, name string) (*inode.Inode, Nfsstat3) {
	buftxn := buftxn.Begin(nfs.txn)
	nfs.glockAcq()

	dip, err1 := nfs.getInodeByFh(buftxn, dfh)
	if err1 != NFS3_OK {
		nfs.glockRel()
		return nil, err1
	}
	ip, err2 := nfs.allocDir(buftxn, dip, name)
	if err2 != NFS3_OK {
		nfs.glockRel()
		return nil, err2
	}

	buftxn.CommitWait(false)
	nfs.glockRel()

	return ip, NFS3_OK
}

func (nfs *BarebonesNfs) OpRmdir(dfh Fh, name string) (Nfsstat3) {
	if name == "." {
		return NFS3ERR_INVAL
	}
	if name == ".." {
		return NFS3ERR_EXIST
	}

	buftxn := buftxn.Begin(nfs.txn)
	nfs.glockAcq()

	dip, err1 := nfs.getInodeByFh(buftxn, dfh)
	if err1 != NFS3_OK {
		nfs.glockRel()
		return err1
	}
	in := nfs.lookupName(dip, name)
	if in == 0 {
		nfs.glockRel()
		return NFS3ERR_NOENT
	}
	ip := nfs.getInode(buftxn, in)
	nfs.freeRecurse(buftxn, ip)
	for i := uint64(0); i < uint64(len(dip.Contents)); i++ {
		if dip.Contents[i] == in {
			dip.Contents[i] = 0
			break
		}
	}
	nfs.writeInode(buftxn, dip)

	buftxn.CommitWait(false)
	nfs.glockRel()

	return NFS3_OK
}

type Entry3 struct {
	Inode  *inode.Inode
	Name   []byte
	Cookie uint64
}

func makeEntry3(ip *inode.Inode, name []byte, cookie uint64) Entry3 {
	return Entry3{
		Inode: ip,
		Name: name,
		Cookie: cookie,
	}
}

func (nfs *BarebonesNfs) OpReadDirPlus(dfh Fh) (*inode.Inode, []Entry3, Nfsstat3) {
	buftxn := buftxn.Begin(nfs.txn)
	dip, err := nfs.getInodeByFh(buftxn, dfh)
	if err != NFS3_OK {
		return nil, nil, err
	}
	var entries []Entry3
	entries = append(entries, makeEntry3(dip, []byte("."), 1))
	if dip.Parent != 0 {
		pip := nfs.getInode(buftxn, dip.Parent)
		entries = append(entries, makeEntry3(pip, []byte(".."), 2))
	}
	for ir := uint64(0); ir < uint64(len(dip.Contents)); ir++ {
		i := uint64(len(dip.Contents)) - ir - 1
		if dip.Contents[i] == 0 {
			continue
		}
		ip := nfs.getInode(buftxn, dip.Contents[i])
		entries = append(entries, makeEntry3(ip, []byte{ dip.Names[i] }, i + 3))
	}
	return dip, entries, NFS3_OK
}
