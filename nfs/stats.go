package nfs

import (
	"io"
	"time"

	"github.com/mit-pdos/go-nfsd/util/stats"
)

const NUM_NFS_OPS = 22

func (nfs *Nfs) recordOp(op uint32, start time.Time) {
	nfs.stats[op].Record(start)
}

var nfsopNames = []string{
	"NULL",
	"GETATTR",
	"SETATTR",
	"LOOKUP",
	"ACCESS",
	"READLINK",
	"READ",
	"WRITE",
	"CREATE",
	"MKDIR",
	"SYMLINK",
	"MKNOD",
	"REMOVE",
	"RMDIR",
	"RENAME",
	"LINK",
	"READDIR",
	"READDIRPLUS",
	"FSSTAT",
	"FSINFO",
	"PATHCONF",
	"COMMIT",
}

func (nfs *Nfs) WriteOpStats(w io.Writer) {
	stats.WriteTable(nfsopNames, nfs.stats[:], w)
}

func (nfs *Nfs) ResetOpStats() {
	for i := range nfs.stats {
		nfs.stats[i].Reset()
	}
}
