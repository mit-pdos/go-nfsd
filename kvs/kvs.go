package kvs

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/buftxn"
	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
)

//
// KVS using txns to implement multiput / multiget transactions
// Keys == Block addresses
//

const DISKSZ uint64 = 10 * 1000
const DISKNAME string = "goose_kvs.img"

type KVS struct {
	super *super.FsSuper
	txn   *txn.Txn
}

type KVPair struct {
	Key uint64
	Val []byte
}

func MkKVS(d disk.Disk) *KVS {
	fsSuper := super.MkFsSuper(d)
	kvs := &KVS{
		super: fsSuper,
		txn:   txn.MkTxn(fsSuper),
	}
	return kvs
}

func (kvs *KVS) MultiPut(pairs []KVPair) bool {
	btxn := buftxn.Begin(kvs.txn)
	for _, p := range pairs {
		akey := addr.MkAddr(p.Key+common.LOGSIZE, 0)
		btxn.OverWrite(akey, common.NBITBLOCK, p.Val)
	}
	ok := btxn.CommitWait(true)
	return ok
}

func (kvs *KVS) Get(key uint64) *KVPair {
	btxn := buftxn.Begin(kvs.txn)
	akey := addr.MkAddr(key+common.LOGSIZE, 0)
	data := btxn.ReadBuf(akey, common.NBITBLOCK).Data
	btxn.CommitWait(true)
	return &KVPair{
		Key: key,
		Val: data,
	}
}

func (kvs *KVS) Delete() {
	kvs.txn.Shutdown()
}
