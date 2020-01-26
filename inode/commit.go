package inode

import (
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/fstxn"
	"github.com/mit-pdos/goose-nfsd/util"
)

func commitWait(op *fstxn.FsTxn, inodes []*Inode, wait bool, abort bool) bool {
	// putInodes may free an inode so must be done before commit
	putInodes(op, inodes)
	return op.CommitWait(wait, abort)
}

func Commit(op *fstxn.FsTxn, inodes []*Inode) bool {
	return commitWait(op, inodes, true, false)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func CommitData(op *fstxn.FsTxn, inodes []*Inode, fh fh.Fh) bool {
	return commitWait(op, inodes, true, false)
}

// Commit transaction, but don't write to stable storage
func CommitUnstable(op *fstxn.FsTxn, inodes []*Inode, fh fh.Fh) bool {
	util.DPrintf(5, "commitUnstable\n")
	if len(inodes) > 1 {
		panic("commitUnstable")
	}
	return commitWait(op, inodes, false, false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func CommitFh(op *fstxn.FsTxn, fh fh.Fh, inodes []*Inode) bool {
	return op.Flush()
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func Abort(op *fstxn.FsTxn, inodes []*Inode) bool {
	util.DPrintf(1, "Abort %v\n", inodes)
	return commitWait(op, inodes, true, true)
}
