package goose_nfs

import (
	"github.com/tchajed/goose/machine"

	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/marshal"
	"github.com/mit-pdos/goose-nfsd/trans"
)

const DIRENTSZ uint64 = 32
const MAXNAMELEN = DIRENTSZ - 8

type dirEnt struct {
	inum fs.Inum
	name string // <= MAXNAMELEN
}

func illegalName(name Filename3) bool {
	n := name
	return n == "." || n == ".."
}

func (dip *inode) lookupName(trans *trans.Trans, name Filename3) (fs.Inum, uint64) {
	if dip.kind != NF3DIR {
		return NULLINUM, 0
	}
	var inum = NULLINUM
	var finalOffset uint64 = 0
	for off := uint64(0); off < dip.size; off += DIRENTSZ {
		data, _ := dip.read(trans, off, DIRENTSZ)
		if uint64(len(data)) != DIRENTSZ {
			break
		}
		de := decodeDirEnt(data)
		if de.inum == NULLINUM {
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

func (dip *inode) addName(trans *trans.Trans, inum fs.Inum, name Filename3) bool {
	var finalOff uint64 = 0

	if dip.kind != NF3DIR || uint64(len(name)) >= MAXNAMELEN {
		return false
	}
	for off := uint64(0); off < dip.size; off += DIRENTSZ {
		data, _ := dip.read(trans, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == NULLINUM {
			finalOff = off
			break
		}
	}
	de := &dirEnt{inum: inum, name: string(name)}
	ent := encodeDirEnt(de)
	n, _ := dip.write(trans, finalOff, DIRENTSZ, ent)
	return n == DIRENTSZ
}

func (dip *inode) remName(trans *trans.Trans, name Filename3) bool {
	inum, off := dip.lookupName(trans, name)
	if inum == NULLINUM {
		return true
	}
	de := &dirEnt{inum: NULLINUM, name: ""}
	ent := encodeDirEnt(de)
	n, _ := dip.write(trans, off, DIRENTSZ, ent)
	return n == DIRENTSZ
}

func (dip *inode) isDirEmpty(trans *trans.Trans) bool {
	var empty bool = true

	// check all entries after . and ..
	for off := uint64(2 * DIRENTSZ); off < dip.size; {
		data, _ := dip.read(trans, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		empty = false
		break
	}
	return empty
}

func (dip *inode) initDir(trans *trans.Trans, parent fs.Inum) bool {
	if !dip.addName(trans, dip.inum, ".") {
		return false
	}
	return dip.addName(trans, parent, "..")
}

func (dip *inode) mkRootDir(trans *trans.Trans) bool {
	if !dip.addName(trans, dip.inum, ".") {
		return false
	}
	return dip.addName(trans, dip.inum, "..")
}

// XXX inode locking order violated
func (dip *inode) ls3(trans *trans.Trans, start Cookie3, dircount Count3) Dirlistplus3 {
	var lst *Entryplus3
	var last *Entryplus3
	var eof bool = true
	var ip *inode
	var begin = uint64(start)
	if begin != 0 {
		begin += DIRENTSZ
	}
	for off := begin; off < dip.size; {
		data, _ := dip.read(trans, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		if de.inum != dip.inum {
			ip = getInodeInum(trans, de.inum)
		} else {
			ip = dip
		}
		fattr := ip.mkFattr()
		fh := &fh{ino: ip.inum, gen: ip.gen}
		ph := Post_op_fh3{Handle_follows: true, Handle: fh.makeFh3()}
		pa := Post_op_attr{Attributes_follow: true, Attributes: fattr}

		// XXX hack release inode and inode block
		if ip != dip {
			ip.put(trans)
			ip.releaseInode(trans)
		}

		e := &Entryplus3{Fileid: Fileid3(de.inum),
			Name:            Filename3(de.name),
			Cookie:          Cookie3(off),
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
		if Count3(off-begin) >= dircount {
			eof = false
			break
		}
	}
	dl := Dirlistplus3{Entries: lst, Eof: eof}
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
