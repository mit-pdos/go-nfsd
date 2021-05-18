package kvs

import (
	"fmt"

	"github.com/mit-pdos/go-journal/addr"
	"github.com/mit-pdos/go-journal/buftxn"
	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-journal/txn"
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
	txn *txn.Txn
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
	txn := txn.MkTxn(d)
	kvs := &KVS{
		sz:  sz,
		txn: txn,
	}
	return kvs
}

func (kvs *KVS) MultiPut(pairs []KVPair) bool {
	btxn := buftxn.Begin(kvs.txn)
	for _, p := range pairs {
		if p.Key >= kvs.sz || p.Key < common.LOGSIZE {
			panic(fmt.Errorf("out-of-bounds put at %v", p.Key))
		}
		akey := addr.MkAddr(p.Key, 0)
		btxn.OverWrite(akey, common.NBITBLOCK, p.Val)
	}
	ok := btxn.CommitWait(true)
	return ok
}

func (kvs *KVS) Get(key uint64) (*KVPair, bool) {
	if key > kvs.sz || key < common.LOGSIZE {
		panic(fmt.Errorf("out-of-bounds get at %v", key))
	}
	btxn := buftxn.Begin(kvs.txn)
	akey := addr.MkAddr(key, 0)
	data := util.CloneByteSlice(btxn.ReadBuf(akey, common.NBITBLOCK).Data)
	ok := btxn.CommitWait(true)
	return &KVPair{
		Key: key,
		Val: data,
	}, ok
}

func (kvs *KVS) Delete() {
	kvs.txn.Shutdown()
}
