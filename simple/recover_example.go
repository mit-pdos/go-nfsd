package simple

import (
	"github.com/tchajed/goose/machine/disk"

	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/mit-pdos/goose-nfsd/lockmap"
	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/mit-pdos/goose-nfsd/txn"
)

func exampleWorker(nfs *Nfs, ino common.Inum) {
	fh := Fh{Ino: ino}
	buf := make([]byte, 1024)
	nfs.NFSPROC3_GETATTR(nfstypes.GETATTR3args{Object: fh.MakeFh3()})
	nfs.NFSPROC3_READ(nfstypes.READ3args{File: fh.MakeFh3(), Offset: 0, Count: 1024})
	nfs.NFSPROC3_WRITE(nfstypes.WRITE3args{File: fh.MakeFh3(), Offset: 0, Count: 1024, Data: buf})
}

func RecoverExample(d disk.Disk) {
	txn := txn.MkTxn(d)
	lockmap := lockmap.MkLockMap()

	nfs := &Nfs{
		t: txn,
		l: lockmap,
	}

	go func() { exampleWorker(nfs, 3) }()
	go func() { exampleWorker(nfs, 3) }()
	go func() { exampleWorker(nfs, 4) }()
}
