package kvs

import (
	"fmt"
	"os"
	"path/filepath"

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
const PRESENTBNUM uint64 = common.LOGSIZE

type KVS struct {
	name  *string
	super *super.FsSuper
	txn   *txn.Txn
}

type KVPair struct {
	Key uint64
	Val []byte
}

func MkKVS() *KVS {
	var tmpdir string
	tmpdir = "/dev/shm"
	f, err := os.Stat(tmpdir)
	if !(err == nil && f.IsDir()) {
		tmpdir = os.TempDir()
	}
	n := filepath.Join(tmpdir, "goose_kvs.img")
	os.Remove(n)
	d, err := disk.NewFileDisk(n, DISKSZ)
	if err != nil {
		panic(fmt.Errorf("could not create file disk: %v", err))
	}

	fsSuper := super.MkFsSuper(d)
	kvs := &KVS{
		name:  &n,
		super: fsSuper,
		txn:   txn.MkTxn(fsSuper),
	}
	return kvs
}

func (kvs *KVS) MultiPut(pairs []KVPair) bool {
	btxn := buftxn.Begin(kvs.txn)
	for _, p := range pairs {
		akey := addr.MkAddr(p.Key+common.LOGSIZE, 0, disk.BlockSize*8)
		btxn.OverWrite(akey, p.Val)
	}
	ok := btxn.CommitWait(true)
	return ok
}

func (kvs *KVS) Get(key uint64) *KVPair {
	btxn := buftxn.Begin(kvs.txn)
	akey := addr.MkAddr(key+common.LOGSIZE, 0, disk.BlockSize*8)
	data := btxn.ReadBuf(akey).Blk
	btxn.CommitWait(true)
	return &KVPair{
		Key: key,
		Val: data,
	}
}

func (kvs *KVS) Delete() {
	kvs.txn.Shutdown()
	if kvs.name != nil {
		err := os.Remove(*kvs.name)
		if err != nil {
			panic(err)
		}
	}
}
