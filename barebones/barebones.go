package barebones

import (
	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/buftxn"
)

type Nfsstat3 uint32

const NFS3_OK Nfsstat3 = 0
const NFS3ERR_PERM Nfsstat3 = 1
const NFS3ERR_NOENT Nfsstat3 = 2
const NFS3ERR_IO Nfsstat3 = 5
const NFS3ERR_NXIO Nfsstat3 = 6
const NFS3ERR_ACCES Nfsstat3 = 13
const NFS3ERR_EXIST Nfsstat3 = 17
const NFS3ERR_XDEV Nfsstat3 = 18
const NFS3ERR_NODEV Nfsstat3 = 19
const NFS3ERR_NOTDIR Nfsstat3 = 20
const NFS3ERR_ISDIR Nfsstat3 = 21
const NFS3ERR_INVAL Nfsstat3 = 22
const NFS3ERR_FBIG Nfsstat3 = 27
const NFS3ERR_NOSPC Nfsstat3 = 28
const NFS3ERR_ROFS Nfsstat3 = 30
const NFS3ERR_MLINK Nfsstat3 = 31
const NFS3ERR_NAMETOOLONG Nfsstat3 = 63
const NFS3ERR_NOTEMPTY Nfsstat3 = 66
const NFS3ERR_DQUOT Nfsstat3 = 69
const NFS3ERR_STALE Nfsstat3 = 70
const NFS3ERR_REMOTE Nfsstat3 = 71
const NFS3ERR_BADHANDLE Nfsstat3 = 10001
const NFS3ERR_NOT_SYNC Nfsstat3 = 10002
const NFS3ERR_BAD_COOKIE Nfsstat3 = 10003
const NFS3ERR_NOTSUPP Nfsstat3 = 10004
const NFS3ERR_TOOSMALL Nfsstat3 = 10005
const NFS3ERR_SERVERFAULT Nfsstat3 = 10006
const NFS3ERR_BADTYPE Nfsstat3 = 10007
const NFS3ERR_JUKEBOX Nfsstat3 = 10008

type BarebonesNfs struct {
	glocks   *lockmap.LockMap // for now, we only use block 0
	fs       *super.FsSuper
	txn      *txn.Txn
	bitmap   []byte
}

type Fh struct {
	Ino common.Inum
	Gen uint64
}

func (nfs *BarebonesNfs) Shutdown() {
	nfs.txn.Shutdown()
}

// global locking
func (nfs *BarebonesNfs) glockAcq() {
	nfs.glocks.Acquire(0, 0)
}

func (nfs *BarebonesNfs) glockRel() {
	nfs.glocks.Release(0, 0)
}

func (nfs *BarebonesNfs) getInode(buftxn *buftxn.BufTxn, inum common.Inum) *inode.Inode {
	// this will never trigger
	// if inum >= nfs.fs.NInode() {
	// 	return nil
	// }
	addr := nfs.fs.Inum2Addr(inum)
	buf := buftxn.ReadBuf(addr, common.INODESZ*8)
	return inode.Decode(buf, inum)
}

func (nfs *BarebonesNfs) GetRootInode() *inode.Inode {
	buftxn := buftxn.Begin(nfs.txn)
	return nfs.getInode(buftxn, common.ROOTINUM)
}

func (nfs *BarebonesNfs) getInodeByFh(buftxn *buftxn.BufTxn, fh Fh) (*inode.Inode, Nfsstat3) {
	ip := nfs.getInode(buftxn, fh.Ino)
	if ip == nil {
		return nil, NFS3ERR_BADHANDLE
	}
	if ip.Gen != fh.Gen {
		return nil, NFS3ERR_STALE
	}
	return ip, NFS3_OK
}

func (nfs *BarebonesNfs) writeInode(buftxn *buftxn.BufTxn, ip *inode.Inode) {
	addr := nfs.fs.Inum2Addr(ip.Inum)
	buftxn.OverWrite(addr, common.INODESZ*8, ip.Encode())
}

func (nfs *BarebonesNfs) lookupName(dip *inode.Inode, name string) common.Inum {
	if name == "." {
		return dip.Inum
	}
	if name == ".." {
		return dip.Parent
	}
	var ip common.Inum
	for i := uint64(0); i < uint64(len(dip.Contents)); i++ {
		if dip.Contents[i] == 0 {
			continue
		}
		if dip.Names[i] == []byte(name)[0] {
			ip = dip.Contents[i]
			break
		}
	}
	return ip
}

func (nfs *BarebonesNfs) allocInode(buftxn *buftxn.BufTxn, dip *inode.Inode) (*inode.Inode, Nfsstat3) {
	var ip *inode.Inode
	for i := uint64(0); i < uint64(len(nfs.bitmap)) * 8; i++ {
		byteNum := i / 8
		bitNum := i % 8
		if (nfs.bitmap[byteNum] & (1 << bitNum)) == 0 {
			nfs.bitmap[byteNum] = nfs.bitmap[byteNum] | (1 << bitNum)
			bitaddr := addr.MkBitAddr(i / common.NBITBLOCK, i % common.NBITBLOCK)
			buftxn.OverWrite(bitaddr, 1, []byte{1 << bitNum})
			ip = nfs.getInode(buftxn, i)
			ip.InitInode(i, dip.Inum)
			nfs.writeInode(buftxn, ip)
			break
		}
	}
	if ip == nil {
		return nil, NFS3ERR_NOSPC
	}
	return ip, NFS3_OK
}

func (nfs *BarebonesNfs) allocDir(buftxn *buftxn.BufTxn, dip *inode.Inode, name string) (*inode.Inode, Nfsstat3) {
	if len(name) == 0 {
		return nil, NFS3ERR_ACCES
	}
	if len(name) > 1 {
		return nil, NFS3ERR_NAMETOOLONG
	}
	existing := nfs.lookupName(dip, name)
	if existing != 0 {
		return nil, NFS3ERR_EXIST
	}
	var ip *inode.Inode
	var err Nfsstat3
	err = NFS3ERR_ACCES
	for i := uint64(0); i < uint64(len(dip.Contents)); i++ {
		if dip.Contents[i] == 0 {

			/* BYPASS ALLOC START */
			// here we allocate inodes statically
			inum := 2 + i
			ip = nfs.getInode(buftxn, inum)
			ip.InitInode(inum, dip.Inum)
			nfs.writeInode(buftxn, ip)
			/* BYPASS ALLOC END */

			// ipAlloc, errAlloc := nfs.allocInode(buftxn, dip)
			// ip = ipAlloc
			// err = errAlloc
			// if err != NFS3_OK {
			// 	break
			// }
			dip.Contents[i] = ip.Inum
			dip.Names[i] = []byte(name)[0]
			nfs.writeInode(buftxn, dip)
			break
		}
	}
	return ip, err
}

func (nfs *BarebonesNfs) freeRecurse(buftxn *buftxn.BufTxn, dip *inode.Inode) {
	for i := uint64(0); i < uint64(len(dip.Contents)); i++ {
		if dip.Contents[i] == 0{
			continue
		}
		// disable recursion for now
		// ip := nfs.getInode(buftxn, dip.Contents[i])
		// nfs.freeRecurse(buftxn, ip)
	}
	byteNum := dip.Inum / 8
	bitNum := dip.Inum % 8
	nfs.bitmap[byteNum] = nfs.bitmap[byteNum] & ^(1 << bitNum)
	bitaddr := addr.MkBitAddr(dip.Inum / common.NBITBLOCK, dip.Inum % common.NBITBLOCK)
	buftxn.OverWrite(bitaddr, 1, []byte{ 0 })
}
