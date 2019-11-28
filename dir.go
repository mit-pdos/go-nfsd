package goose_nfs

import (
	"github.com/tchajed/goose/machine/disk"
)

const MaxNameLen = 4096 - 1 - 8

type DirEnt struct {
	Valid bool
	Name  string // max 4096-1-8=4087 bytes
	Inum  Inum
}

func encodeDirEnt(de *DirEnt, blk disk.Block) disk.Block {
	if len(de.Name) > MaxNameLen {
		panic("directory entry name too long")
	}
	enc := NewEnc(blk)
	enc.PutString(de.Name)
	enc.PutBool(de.Valid)
	enc.PutInt(de.Inum)
	return enc.Finish()
}

func decodeDirEnt(b disk.Block) *DirEnt {
	dec := NewDec(b)
	de := &DirEnt{}
	de.Name = dec.GetString()
	de.Valid = dec.GetBool()
	de.Inum = dec.GetInt()
	return de
}

func (ip *Inode) lookupLink(txn *Txn, name Filename3) uint64 {
	if ip.kind != NF3DIR {
		return 0
	}
	blocks := ip.size / disk.BlockSize
	for b := uint64(0); b < blocks; b++ {
		blk := (*txn).Read(ip.blks[b])
		de := decodeDirEnt(blk)
		if !de.Valid {
			continue
		}
		if de.Name == string(name) {
			return de.Inum
		}
	}
	return 0
}

func writeLink(blk disk.Block, txn *Txn, inum uint64, name Filename3, blkno uint64) bool {
	de := &DirEnt{Valid: true, Inum: inum, Name: string(name)}
	encodeDirEnt(de, blk)
	ok := (*txn).Write(blkno, blk)
	return ok
}

func delLink(blk disk.Block, txn *Txn, blkno uint64) bool {
	de := &DirEnt{Valid: false, Inum: NULLINUM, Name: string("")}
	encodeDirEnt(de, blk)
	ok := (*txn).Write(blkno, blk)
	return ok
}

func (dip *Inode) addLink(txn *Txn, inum uint64, name Filename3) bool {
	var freede *DirEnt

	if dip.kind != NF3DIR {
		return false
	}
	blocks := dip.size / disk.BlockSize
	for b := uint64(0); b < blocks; b++ {
		blk := (*txn).Read(dip.blks[b])
		de := decodeDirEnt(blk)
		if !de.Valid {
			writeLink(blk, txn, inum, name, dip.blks[b])
			freede = de
			break
		}
		continue
	}
	if freede != nil {
		return true
	}
	ok := dip.resize(txn, dip.size+disk.BlockSize)
	if !ok {
		return false
	}
	blk := (*txn).Read(dip.blks[blocks])
	writeLink(blk, txn, inum, name, dip.blks[blocks])
	if !ok {
		panic("addLink")
	}
	return true
}

func (dip *Inode) remLink(txn *Txn, name Filename3) Inum {
	var inum Inum = NULLINUM
	if dip.kind != NF3DIR {
		return NULLINUM
	}
	blocks := dip.size / disk.BlockSize
	for b := uint64(0); b < blocks; b++ {
		blk := (*txn).Read(dip.blks[b])
		de := decodeDirEnt(blk)
		if de.Valid && de.Name == string(name) {
			inum = de.Inum
			delLink(blk, txn, dip.blks[b])
			break
		}
		continue
	}
	if inum == NULLINUM {
		return NULLINUM
	}
	return inum
}

// XXX . and ..
func (dip *Inode) dirEmpty(txn *Txn) bool {
	return dip.size == 0
}
