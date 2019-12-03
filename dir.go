package goose_nfs

import (
	"log"
)

const DIRENTSZ = 32
const MAXNAMELEN = DIRENTSZ - 8

type DirEnt struct {
	Inum Inum
	Name string // <= MAXNAMELEN
}

func illegalName(name Filename3) bool {
	n := string(name)
	return n == "." || n == ".."
}

func (dip *Inode) lookupName(txn *Txn, name Filename3) (Inum, uint64) {
	if dip.kind != NF3DIR {
		return NULLINUM, 0
	}
	for off := uint64(0); off < dip.size; {
		data, _ := dip.read(txn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.Inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		if de.Name == string(name) {
			return de.Inum, off
		}
		off = off + DIRENTSZ
	}
	return NULLINUM, 0
}

func (dip *Inode) addName(txn *Txn, inum uint64, name Filename3) bool {
	var fail bool = false
	var off uint64 = 0

	if dip.kind != NF3DIR {
		return false
	}
	for off = uint64(0); off < dip.size; {
		data, _ := dip.read(txn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.Inum == NULLINUM {
			break
		}
		off = off + DIRENTSZ
		continue
	}
	if fail {
		return false
	}
	de := &DirEnt{Inum: inum, Name: string(name)}
	ent := encodeDirEnt(de)
	_, ok := dip.write(txn, off, DIRENTSZ, ent)
	return ok
}

func (dip *Inode) remName(txn *Txn, name Filename3) bool {
	inum, off := dip.lookupName(txn, name)
	if inum == NULLINUM {
		return true
	}
	de := &DirEnt{Inum: NULLINUM, Name: ""}
	ent := encodeDirEnt(de)
	_, ok := dip.write(txn, off, DIRENTSZ, ent)
	return ok
}

func (dip *Inode) isDirEmpty(txn *Txn) bool {
	var empty bool = true
	for off := uint64(2 * DIRENTSZ); off < dip.size; {
		data, _ := dip.read(txn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.Inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		empty = false
		break
	}
	return empty
}

func (dip *Inode) mkdir(txn *Txn, parent Inum) bool {
	if !dip.addName(txn, dip.inum, ".") {
		return false
	}
	if !dip.addName(txn, parent, "..") {
		return false
	}
	return true
}

func (dip *Inode) mkRootDir(txn *Txn) bool {
	if !dip.addName(txn, dip.inum, ".") {
		return false
	}
	if !dip.addName(txn, dip.inum, "..") {
		return false
	}
	return true
}

func (dip *Inode) ls(txn *Txn, count Count3) Dirlist3 {
	var lst *Entry3
	for off := uint64(0); off < dip.size; {
		data, _ := dip.read(txn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.Inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		e := &Entry3{Fileid: Fileid3(de.Inum),
			Name:      Filename3(de.Name),
			Cookie:    Cookie3(0),
			Nextentry: lst,
		}
		lst = e
		off = off + DIRENTSZ
	}
	dl := Dirlist3{Entries: lst, Eof: true}
	return dl
}

// XXX inode locking order violated
func (dip *Inode) ls3(txn *Txn, start Cookie3, dircount Count3) Dirlistplus3 {
	var lst *Entryplus3
	var last *Entryplus3
	var eof bool = true
	var ip *Inode
	log.Printf("%d: start: %v\n", dip.inum, start)
	for off := uint64(start); off < dip.size; {
		data, _ := dip.read(txn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.Inum == NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		if de.Inum != dip.inum {
			ip = getInodeInum(txn, de.Inum)
		} else {
			ip = dip
		}
		fattr := ip.mkFattr()
		fh := &Fh{ino: ip.inum, gen: ip.gen}
		ph := Post_op_fh3{Handle_follows: true, Handle: fh.makeFh3()}
		pa := Post_op_attr{Attributes_follow: true, Attributes: fattr}

		// XXX hack release inode and inode block
		if ip != dip {
			ip.put(txn)
			ip.unlock()
			txn.fs.releaseInodeBlock(txn, ip.inum)
		}

		e := &Entryplus3{Fileid: Fileid3(de.Inum),
			Name:            Filename3(de.Name),
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
		if Count3(off-uint64(start)) >= dircount {
			eof = false
			break
		}
	}
	dl := Dirlistplus3{Entries: lst, Eof: eof}
	return dl
}
