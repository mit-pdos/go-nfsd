package kvs

import (
	"fmt"

	"github.com/mit-pdos/go-journal/addr"
	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-journal/jrnl"
	"github.com/mit-pdos/go-journal/obj"
	"github.com/mit-pdos/go-journal/util"
	"github.com/tchajed/goose/machine/disk"
)

//
// KVS using txns to implement multiput / multiget transactions
// Keys == Block addresses
//

const DISKNAME string = "goose_kvs.img"

type KVS struct {
	sz  uint64
	log *obj.Log
}

type KVPair struct {
	Key uint64
	Val []byte
}

func MkKVS(d disk.Disk, sz uint64) *KVS {
	/*if sz > d.Size() {
		panic("kvs larger than disk")
	}*/
	// XXX just need to assume that the kvs is less than the disk size?
	log := obj.MkLog(d)
	kvs := &KVS{
		sz:  sz,
		log: log,
	}
	return kvs
}

func (kvs *KVS) MultiPut(pairs []KVPair) bool {
	op := jrnl.Begin(kvs.log)
	for _, p := range pairs {
		if p.Key >= kvs.sz || p.Key < common.LOGSIZE {
			panic(fmt.Errorf("out-of-bounds put at %v", p.Key))
		}
		akey := addr.MkAddr(p.Key, 0)
		op.OverWrite(akey, common.NBITBLOCK, p.Val)
	}
	ok := op.CommitWait(true)
	return ok
}

func (kvs *KVS) Get(key uint64) (*KVPair, bool) {
	if key > kvs.sz || key < common.LOGSIZE {
		panic(fmt.Errorf("out-of-bounds get at %v", key))
	}
	op := jrnl.Begin(kvs.log)
	akey := addr.MkAddr(key, 0)
	data := util.CloneByteSlice(op.ReadBuf(akey, common.NBITBLOCK).Data)
	ok := op.CommitWait(true)
	return &KVPair{
		Key: key,
		Val: data,
	}, ok
}

func (kvs *KVS) Delete() {
	kvs.log.Shutdown()
}
