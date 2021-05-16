package goose_nfs

import (
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/dir"
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

func Ls3(dip *inode.Inode, op *fstxn.FsTxn, start nfstypes.Cookie3, dircount, maxcount nfstypes.Count3) nfstypes.Dirlistplus3 {
	var lst *nfstypes.Entryplus3
	var last *nfstypes.Entryplus3
	eof := dir.Apply(dip, op, uint64(start), uint64(dircount), uint64(maxcount),
		func(ip *inode.Inode, name string, inum common.Inum, off uint64) {
			fattr := ip.MkFattr()
			fh := &fh.Fh{Ino: ip.Inum, Gen: ip.Gen}
			ph := nfstypes.Post_op_fh3{
				Handle_follows: true,
				Handle:         fh.MakeFh3(),
			}
			pa := nfstypes.Post_op_attr{
				Attributes_follow: true,
				Attributes:        fattr,
			}
			e := &nfstypes.Entryplus3{
				Fileid:          nfstypes.Fileid3(inum),
				Name:            nfstypes.Filename3(name),
				Cookie:          nfstypes.Cookie3(off),
				Name_attributes: pa,
				Name_handle:     ph,
				Nextentry:       nil,
			}
			if last == nil {
				lst = e
				last = e
			} else {
				last.Nextentry = e
				last = e
			}
		})
	dl := nfstypes.Dirlistplus3{Entries: lst, Eof: eof}
	return dl
}

func Readdir3(dip *inode.Inode, op *fstxn.FsTxn,
	start nfstypes.Cookie3, count nfstypes.Count3) nfstypes.Dirlist3 {
	var lst *nfstypes.Entry3
	var last *nfstypes.Entry3
	eof := dir.ApplyEnts(dip, op, uint64(start), uint64(count),
		func(name string, inum common.Inum, off uint64) {
			e := &nfstypes.Entry3{
				Fileid:    nfstypes.Fileid3(inum),
				Name:      nfstypes.Filename3(name),
				Cookie:    nfstypes.Cookie3(off),
				Nextentry: nil,
			}
			if last == nil {
				lst = e
				last = e
			} else {
				last.Nextentry = e
				last = e
			}
		})
	dl := nfstypes.Dirlist3{Entries: lst, Eof: eof}
	return dl
}
