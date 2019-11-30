package goose_nfs

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
		data, _, ok := dip.read(txn, off, DIRENTSZ)
		if !ok {
			// XXX return false?
			panic("lookupName")
			break
		}
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
		data, _, ok := dip.read(txn, off, DIRENTSZ)
		if !ok {
			fail = true
			break
		}
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
		data, _, ok := dip.read(txn, off, DIRENTSZ)
		if !ok {
			panic("isDirEmpty")
		}
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
