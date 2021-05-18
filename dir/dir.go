package dir

import (
	"github.com/tchajed/marshal"

	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-journal/util"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/inode"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

const DIRENTSZ uint64 = 128
const MAXNAMELEN = DIRENTSZ - 16 // uint64 for inum + uint64 for len(name)

type dirEnt struct {
	inum common.Inum
	name string // <= MAXNAMELEN
}

func IllegalName(name nfstypes.Filename3) bool {
	n := name
	return n == "." || n == ".."
}

func ScanName(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) (common.Inum, uint64) {
	if dip.Kind != nfstypes.NF3DIR {
		return common.NULLINUM, 0
	}
	var inum = common.NULLINUM
	var finalOffset uint64 = 0
	for off := uint64(0); off < dip.Size; off += DIRENTSZ {
		data, _ := dip.Read(op.Atxn, off, DIRENTSZ)
		if uint64(len(data)) != DIRENTSZ {
			break
		}
		de := decodeDirEnt(data)
		if de.inum == common.NULLINUM {
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

func AddNameDir(dip *inode.Inode, op *fstxn.FsTxn, inum common.Inum,
	name nfstypes.Filename3, lastoff uint64) (uint64, bool) {
	var finalOff uint64

	for off := uint64(lastoff); off < dip.Size; off += DIRENTSZ {
		data, _ := dip.Read(op.Atxn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == common.NULLINUM {
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
	n, _ := dip.Write(op.Atxn, finalOff, DIRENTSZ, ent)
	return finalOff, n == DIRENTSZ
}

func RemNameDir(dip *inode.Inode, op *fstxn.FsTxn, name nfstypes.Filename3) (uint64, bool) {
	inum, off := LookupName(dip, op, name)
	if inum == common.NULLINUM {
		return 0, false
	}
	util.DPrintf(5, "RemNameDir # %v: %v %v off %d\n", dip.Inum, name, inum, off)
	de := &dirEnt{inum: common.NULLINUM, name: ""}
	ent := encodeDirEnt(de)
	n, _ := dip.Write(op.Atxn, off, DIRENTSZ, ent)
	return off, n == DIRENTSZ
}

func IsDirEmpty(dip *inode.Inode, op *fstxn.FsTxn) bool {
	var empty bool = true

	// check all entries after . and ..
	for off := uint64(2 * DIRENTSZ); off < dip.Size; {
		data, _ := dip.Read(op.Atxn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		if de.inum == common.NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		empty = false
		break
	}
	util.DPrintf(10, "IsDirEmpty: %v -> %v\n", dip, empty)
	return empty
}

func InitDir(dip *inode.Inode, op *fstxn.FsTxn, parent common.Inum) bool {
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

const fattr3XDRsize uint64 = 4 + 4 + 4 + // type, mode, nlink
	4 + 4 + // uid, gid
	8 + 8 + // size, used
	8 + // rdev (specdata3)
	8 + 8 + // fsid, fileid
	(3 * 8) // atime, mtime, ctime

// best estimate of entryplus3 size, excluding name
const entryplus3Baggage uint64 = 8 + // fileid
	4 + // name length
	8 + // cookie
	4 + fattr3XDRsize + // post_op_attr header + fattr3
	16 + // name_handle
	8 // pointer

// XXX inode locking order violated
func Apply(dip *inode.Inode, op *fstxn.FsTxn, start uint64,
	dircount uint64, maxcount uint64,
	f func(*inode.Inode, string, common.Inum, uint64)) bool {
	var eof bool = true
	var ip *inode.Inode
	var begin = uint64(start)
	if begin != 0 {
		begin += DIRENTSZ
	}
	// TODO: arbitrary estimate of constant XDR overhead
	var n uint64 = uint64(64)
	var dirbytes uint64 = uint64(0)
	for off := begin; off < dip.Size; {
		data, _ := dip.Read(op.Atxn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		util.DPrintf(5, "Apply: # %v %v off %d\n", dip.Inum, de, off)
		if de.inum == common.NULLINUM {
			off = off + DIRENTSZ
			continue
		}

		// Lock inode, if this transaction doesn't own it already
		var own bool = false
		if op.OwnInum(de.inum) {
			own = true
			ip = op.GetInodeUnlocked(de.inum)
		} else {
			ip = op.GetInodeInum(de.inum)

		}

		f(ip, de.name, de.inum, off)

		// Release inode early, if this trans didn't own it before.
		if !own {
			op.ReleaseInode(ip)
		}

		off = off + DIRENTSZ
		// TODO: unclear what dircount is supposed to included so we pad it with
		// 8 bytes per entry
		dirbytes += uint64(8 + len(de.name))
		n += entryplus3Baggage + uint64(len(de.name))
		if dirbytes >= dircount || n >= maxcount {
			eof = false
			break
		}
	}
	return eof
}

func ApplyEnts(dip *inode.Inode, op *fstxn.FsTxn, start uint64, count uint64,
	f func(string, common.Inum, uint64)) bool {
	var eof bool = true
	var begin = uint64(start)
	if begin != 0 {
		begin += DIRENTSZ
	}
	// TODO: this is supposed to track the size of the XDR-encoded reply in
	// bytes, and we somewhat arbitrarily use 64 as the constant overhead
	var n uint64 = uint64(64)
	for off := begin; off < dip.Size; {
		data, _ := dip.Read(op.Atxn, off, DIRENTSZ)
		de := decodeDirEnt(data)
		util.DPrintf(5, "Apply: # %v %v off %d\n", dip.Inum, de, off)
		if de.inum == common.NULLINUM {
			off = off + DIRENTSZ
			continue
		}
		f(de.name, de.inum, off)

		off = off + DIRENTSZ
		// TODO: estimate of XDR overhead, 16-byte file id, name, cookie, and
		// pointer for linked list
		n += uint64(16 + len(de.name) + 8 + 8)
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
		inum: common.Inum(inum),
		name: name,
	}
}
