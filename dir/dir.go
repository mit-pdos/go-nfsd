package dir

import (
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/goose-nfsd/fs"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/util"
)

const DIRENTSZ uint64 = 128
const MAXNAMELEN = DIRENTSZ - 16 // uint64 for inum + uint64 for len(name)

type dir inode.Inode

type dirEnt struct {
	inum fs.Inum
	name string // <= MAXNAMELEN
}

func IllegalName(name nfstypes.Filename3) bool {
	n := name
	return n == "." || n == ".."
}

func ScanName(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) (fs.Inum, uint64) {
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

func AddNameDir(dip *inode.Inode, op *fstxn.FsTxn, inum fs.Inum,
	name nfstypes.Filename3, lastoff uint64) (uint64, bool) {
	var finalOff uint64

	for off := uint64(lastoff); off < dip.Size; off += DIRENTSZ {
		data, _ := dip.Read(op, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == fs.NULLINUM {
			finalOff = off
			break
		}
	}
	if finalOff == 0 {
		finalOff = dip.Size
	}
	de := &dirEnt{inum: inum, name: string(name)}
	ent := encodeDirEnt(de)
	util.DPrintf(5, "AddNameDir # %v: %v %v %v off %d\n", dip.Inum, name, de, ent, finalOff)
	n, _ := dip.Write(op, finalOff, DIRENTSZ, ent)
	return finalOff, n == DIRENTSZ
}

func RemNameDir(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) (uint64, bool) {
	inum, off := LookupName(dip, op, name)
	if inum == fs.NULLINUM {
		return 0, false
	}
	util.DPrintf(5, "RemNameDir # %v: %v %v off %d\n", dip.Inum, name, inum, off)
	de := &dirEnt{inum: fs.NULLINUM, name: ""}
	ent := encodeDirEnt(de)
	n, _ := dip.Write(op, off, DIRENTSZ, ent)
	return off, n == DIRENTSZ
}

func IsDirEmpty(dip *inode.Inode, op *fstxn.FsTxn) bool {
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
	util.DPrintf(10, "IsDirEmpty: %v -> %v\n", dip, empty)
	return empty
}

func InitDir(dip *inode.Inode, op *fstxn.FsTxn, parent fs.Inum) bool {
	if !AddName(dip, op, dip.Inum, ".") {
		return false
	}
	return AddName(dip, op, parent, "..")
}

func MkRootDir(dip *inode.Inode, op *fstxn.FsTxn) bool {
	if !AddName(dip, op, dip.Inum, ".") {
		return false
	}
	return AddName(dip, op, dip.Inum, "..")
}

// XXX inode locking order violated
func Apply(dip *inode.Inode, op *fstxn.FsTxn, start uint64, count uint64,
	f func(*inode.Inode, string, fs.Inum, uint64)) bool {
	var eof bool = true
	var ip *inode.Inode
	var begin = uint64(start)
	if begin != 0 {
		begin += DIRENTSZ
	}
	var n uint64 = uint64(0)
	for off := begin; off < dip.Size; {
		data, _ := dip.Read(op, off, DIRENTSZ)
		de := decodeDirEnt(data)
		util.DPrintf(5, "Apply: # %v %v off %d\n", dip.Inum, de, off)
		if de.inum == fs.NULLINUM {
			off = off + DIRENTSZ
			continue
		}

		// Lock inode, if this transaction doesn't own it already
		var own bool = false
		if inode.OwnInode(op, de.inum) {
			own = true
			ip = inode.GetInode(op, de.inum)
		} else {
			ip = inode.GetInodeInum(op, de.inum)

		}

		f(ip, de.name, de.inum, off)

		// Put and release inode early, if this trans didn't
		// own it before.
		ip.Put(op)
		if !own {
			ip.ReleaseInode(op)
		}

		off = off + DIRENTSZ
		n += uint64(16 + len(de.name)) // XXX first 3 entries of Entryplus3
		if n >= count {
			eof = false
			break
		}
	}
	return eof
}

// Caller must ensure de.Name fits
func encodeDirEnt(de *dirEnt) []byte {
	enc := marshal.NewEnc(DIRENTSZ)
	enc.PutInt(uint64(de.inum))
	enc.PutInt(uint64(len(de.name)))
	enc.PutBytes([]byte(de.name))
	return enc.Finish()
}

func decodeDirEnt(d []byte) *dirEnt {
	dec := marshal.NewDec(d)
	inum := dec.GetInt()
	l := dec.GetInt()
	name := string(dec.GetBytes(l))
	return &dirEnt{
		inum: fs.Inum(inum),
		name: name,
	}
}
