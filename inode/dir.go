package inode

import (
	"github.com/tchajed/goose/machine"

	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/marshal"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

const DIRENTSZ uint64 = 32
const MAXNAMELEN = DIRENTSZ - 8

type dirEnt struct {
	inum fs.Inum
	name string // <= MAXNAMELEN
}

func IllegalName(name nfstypes.Filename3) bool {
	n := name
	return n == "." || n == ".."
}

func (dip *Inode) LookupName(op *fstxn.FsTxn, name nfstypes.Filename3) (fs.Inum, uint64) {
	if dip.Kind != nfstypes.NF3DIR {
		return fs.NULLINUM, 0
	}
	var inum = fs.NULLINUM
	var finalOffset uint64 = 0
	for off := uint64(0); off < dip.Size; off += DIRENTSZ {
		data, _ := dip.Read(op, off, DIRENTSZ)
		if uint64(len(data)) != DIRENTSZ {
			break
		}
		de := decodeDirEnt(data)
		if de.inum == fs.NULLINUM {
			continue
		}
		if de.name == string(name) {
			inum = de.inum
			finalOffset = off
			break
		}
	}
	return inum, finalOffset
}

func (dip *Inode) AddName(op *fstxn.FsTxn, inum fs.Inum, name nfstypes.Filename3) bool {
	var finalOff uint64 = 0

	if dip.Kind != nfstypes.NF3DIR || uint64(len(name)) >= MAXNAMELEN {
		return false
	}
	for off := uint64(0); off < dip.Size; off += DIRENTSZ {
		data, _ := dip.Read(op, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == fs.NULLINUM {
			finalOff = off
			break
		}
	}
	de := &dirEnt{inum: inum, name: string(name)}
	ent := encodeDirEnt(de)
	n, _ := dip.Write(op, finalOff, DIRENTSZ, ent)
	return n == DIRENTSZ
}

func (dip *Inode) RemName(op *fstxn.FsTxn, name nfstypes.Filename3) bool {
	inum, off := dip.LookupName(op, name)
	if inum == fs.NULLINUM {
		return true
	}
	de := &dirEnt{inum: fs.NULLINUM, name: ""}
	ent := encodeDirEnt(de)
	n, _ := dip.Write(op, off, DIRENTSZ, ent)
	return n == DIRENTSZ
}

func (dip *Inode) IsDirEmpty(op *fstxn.FsTxn) bool {
	var empty bool = true

	// check all entries after . and ..
	for off := uint64(2 * DIRENTSZ); off < dip.Size; {
		data, _ := dip.Read(op, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == fs.NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		empty = false
		break
	}
	return empty
}

func (dip *Inode) InitDir(op *fstxn.FsTxn, parent fs.Inum) bool {
	if !dip.AddName(op, dip.Inum, ".") {
		return false
	}
	return dip.AddName(op, parent, "..")
}

func (dip *Inode) MkRootDir(op *fstxn.FsTxn) bool {
	if !dip.AddName(op, dip.Inum, ".") {
		return false
	}
	return dip.AddName(op, dip.Inum, "..")
}

// XXX inode locking order violated
func (dip *Inode) Ls3(op *fstxn.FsTxn, start nfstypes.Cookie3, dircount nfstypes.Count3) nfstypes.Dirlistplus3 {
	var lst *nfstypes.Entryplus3
	var last *nfstypes.Entryplus3
	var eof bool = true
	var ip *Inode
	var begin = uint64(start)
	if begin != 0 {
		begin += DIRENTSZ
	}
	for off := begin; off < dip.Size; {
		data, _ := dip.Read(op, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == fs.NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		if de.inum != dip.Inum {
			ip = GetInodeInum(op, de.inum)
		} else {
			ip = dip
		}
		fattr := ip.MkFattr()
		fh := &fh.Fh{Ino: ip.Inum, Gen: ip.Gen}
		ph := nfstypes.Post_op_fh3{Handle_follows: true, Handle: fh.MakeFh3()}
		pa := nfstypes.Post_op_attr{Attributes_follow: true, Attributes: fattr}

		// XXX hack release inode and inode block
		if ip != dip {
			ip.put(op)
			ip.releaseInode(op)
		}

		e := &nfstypes.Entryplus3{Fileid: nfstypes.Fileid3(de.inum),
			Name:            nfstypes.Filename3(de.name),
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
		off = off + DIRENTSZ
		if nfstypes.Count3(off-begin) >= dircount {
			eof = false
			break
		}
	}
	dl := nfstypes.Dirlistplus3{Entries: lst, Eof: eof}
	return dl
}

// Caller must ensure de.Name fits
func encodeDirEnt(de *dirEnt) []byte {
	d := make([]byte, DIRENTSZ)
	machine.UInt64Put(d[:8], uint64(de.inum))
	machine.UInt64Put(d[8:16], uint64(len(de.name)))
	marshal.PutBytes(d[16:], []byte(de.name))
	return d
}

func decodeDirEnt(d []byte) *dirEnt {
	de := &dirEnt{
		inum: 0,
		name: "",
	}
	de.inum = fs.Inum(machine.UInt64Get(d[:8]))
	l := machine.UInt64Get(d[8:16])
	de.name = string(d[16 : 16+l])
	return de
}
