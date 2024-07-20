package simple

import (
	"github.com/goose-lang/primitive/disk"

	"github.com/mit-pdos/go-journal/common"
	"github.com/mit-pdos/go-nfsd/nfstypes"
)

func exampleWorker(nfs *Nfs, ino common.Inum) {
	fh := Fh{Ino: ino}
	buf := make([]byte, 1024)
	nfs.NFSPROC3_GETATTR(nfstypes.GETATTR3args{Object: fh.MakeFh3()})
	nfs.NFSPROC3_READ(nfstypes.READ3args{File: fh.MakeFh3(), Offset: 0, Count: 1024})
	nfs.NFSPROC3_WRITE(nfstypes.WRITE3args{File: fh.MakeFh3(), Offset: 0, Count: 1024, Data: buf})
	return
}

func RecoverExample(d disk.Disk) {
	nfs := Recover(d)
	go func() { exampleWorker(nfs, 3) }()
	go func() { exampleWorker(nfs, 3) }()
	go func() { exampleWorker(nfs, 4) }()
}
