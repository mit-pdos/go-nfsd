package kvs

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/buf"
	"github.com/mit-pdos/goose-nfsd/super"
	"github.com/mit-pdos/goose-nfsd/txn"
)

//
// KVS using txns to implement multiput / multiget transactions
// Keys == Block addresses
//

const DISKSZ uint64 = 10 * 1000

type KVS struct {
	super       *super.FsSuper
	txn         *txn.Txn
	presentBufs *buf.BufMap
}

type KVPair struct {
	Key addr.Addr
	Val []byte
}

func MkKVS() *KVS {
	r := rand.Uint64()
	tmpdir := "/dev/shm"
	f, err := os.Stat(tmpdir)
	if !(err == nil && f.IsDir()) {
		tmpdir = os.TempDir()
	}
	n := filepath.Join(tmpdir, "goose"+strconv.FormatUint(r, 16)+".img")
	d, err := disk.NewFileDisk(n, DISKSZ)
	if err != nil {
		panic(fmt.Errorf("could not create file disk: %v", err))
	}

	fsSuper := super.MkFsSuper(d)
	kvs := &KVS{
		presentBufs: buf.MkBufMap(),
		super:       fsSuper,
		txn:         txn.MkTxn(fsSuper),
	}
	return kvs
}

func (kvs *KVS) MultiPut(pairs []KVPair) bool {
	var bufs []*buf.Buf
	for _, p := range pairs {
		b := kvs.presentBufs.Lookup(p.Key)
		if b == nil {
			b = kvs.txn.Load(p.Key)
			kvs.presentBufs.Insert(b)
		}
		if uint64(len(p.Val)*8) != b.Addr.Sz {
			panic("overwrite")
		}
		b.Blk = p.Val
		bufs = append(bufs, b)
	}
	ok := kvs.txn.CommitWait(bufs, true, kvs.txn.GetTransId())
	return ok
}

func (kvs *KVS) Get(key addr.Addr) *KVPair {
	// only return a key if it has been added to the map
	// otherwise return nil
	var data []byte
	b := kvs.presentBufs.Lookup(key)
	if b != nil {
		data = b.Blk
	}
	return &KVPair{
		Key: key,
		Val: data,
	}
}
