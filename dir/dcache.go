package dir

import (
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/dcache"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

func mkDcache(dip *inode.Inode, op *fstxn.FsTxn) {
	dip.Dcache = dcache.MkDcache()
	Apply(dip, op, 0, dip.Size, 100000000,
		func(ip *inode.Inode, name string, inum common.Inum, off uint64) {
			dip.Dcache.Add(name, inum, off)
		})
}

func LookupName(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) (common.Inum, uint64) {
	if dip.Kind != nfstypes.NF3DIR {
		return common.NULLINUM, 0
	}
	var inum = common.NULLINUM
	var finalOffset uint64 = 0
	if dip.Dcache == nil {
		mkDcache(dip, op)
	}
	dentry, ok := dip.Dcache.Lookup(string(name))
	if ok {
		inum = dentry.Inum
		finalOffset = dentry.Off
	}
	return inum, finalOffset
}

func AddName(dip *inode.Inode, op *fstxn.FsTxn, inum common.Inum, name nfstypes.Filename3) bool {
	if dip.Kind != nfstypes.NF3DIR || uint64(len(name)) >= MAXNAMELEN {
		return false
	}
	if dip.Dcache == nil {
		mkDcache(dip, op)
	}
	off, ok := AddNameDir(dip, op, inum, name, dip.Dcache.Lastoff)
	if ok {
		dip.Dcache.Lastoff = off
		dip.Dcache.Add(string(name), inum, off)
	}
	return ok
}

func RemName(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) bool {
	if dip.Kind != nfstypes.NF3DIR || uint64(len(name)) >= MAXNAMELEN {
		return false
	}
	if dip.Dcache == nil {
		mkDcache(dip, op)
	}
	off, ok := RemNameDir(dip, op, name)
	if ok {
		dip.Dcache.Lastoff = off
		ok := dip.Dcache.Del(string(name))
		if !ok {
			panic("RemName")
		}
		return true
	}
	return false
}
