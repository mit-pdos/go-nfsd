package goose_nfs

import (
	"sync/atomic"
	"time"

	"github.com/mit-pdos/goose-nfsd/nfstypes"
)

const NUM_NFS_OPS = 22

func (nfs *Nfs) incCount(op uint32) {
	atomic.AddUint32(&nfs.opCounts[op], 1)
}

func (nfs *Nfs) incTime(op uint32, nsec uint64) {
	atomic.AddUint64(&nfs.opNanos[op], nsec)
}

func (nfs *Nfs) recordOp(op uint32, start time.Time) {
	nfs.incCount(op)
	dur := time.Now().Sub(start)
	nfs.incTime(op, uint64(dur.Nanoseconds()))
}

var nfsopNames = map[uint32]string{
	nfstypes.NFSPROC3_NULL:        "NULL",
	nfstypes.NFSPROC3_GETATTR:     "GETATTR",
	nfstypes.NFSPROC3_SETATTR:     "SETATTR",
	nfstypes.NFSPROC3_LOOKUP:      "LOOKUP",
	nfstypes.NFSPROC3_ACCESS:      "ACCESS",
	nfstypes.NFSPROC3_READLINK:    "READLINK",
	nfstypes.NFSPROC3_READ:        "READ",
	nfstypes.NFSPROC3_WRITE:       "WRITE",
	nfstypes.NFSPROC3_CREATE:      "CREATE",
	nfstypes.NFSPROC3_MKDIR:       "MKDIR",
	nfstypes.NFSPROC3_SYMLINK:     "SYMLINK",
	nfstypes.NFSPROC3_MKNOD:       "MKNOD",
	nfstypes.NFSPROC3_REMOVE:      "REMOVE",
	nfstypes.NFSPROC3_RMDIR:       "RMDIR",
	nfstypes.NFSPROC3_RENAME:      "RENAME",
	nfstypes.NFSPROC3_LINK:        "LINK",
	nfstypes.NFSPROC3_READDIR:     "READDIR",
	nfstypes.NFSPROC3_READDIRPLUS: "READDIRPLUS",
	nfstypes.NFSPROC3_FSSTAT:      "FSSTAT",
	nfstypes.NFSPROC3_FSINFO:      "FSINFO",
	nfstypes.NFSPROC3_PATHCONF:    "PATHCONF",
	nfstypes.NFSPROC3_COMMIT:      "COMMIT",
}

type OpCount struct {
	Op        string
	Count     uint32
	TimeNanos uint64
}

func (nfs *Nfs) GetOpStats() []OpCount {
	// get all counts locally with atomic reads
	counts := make([]uint32, NUM_NFS_OPS)
	times := make([]uint64, NUM_NFS_OPS)
	for i := range counts {
		counts[i] = atomic.LoadUint32(&nfs.opCounts[i])
		times[i] = atomic.LoadUint64(&nfs.opNanos[i])
	}

	var stats []OpCount
	for op := range counts {
		count := counts[op]
		time := times[op]
		stats = append(stats, OpCount{
			Op:        nfsopNames[uint32(op)],
			Count:     count,
			TimeNanos: time,
		})
	}
	return stats
}
