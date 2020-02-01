package inode

import (
	"github.com/mit-pdos/goose-nfsd/fh"
)

func commitWait(op *FsTxn, inodes []*Inode, wait bool, abort bool) bool {
	// putInodes may free an inode so must be done before commit
	putInodes(op)
	ok := op.buftxn.CommitWait(wait, abort)
	releaseInodes(op)
	return ok
}

func Commit(op *FsTxn, inodes []*Inode) bool {
	return commitWait(op, inodes, true, false)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func CommitData(op *FsTxn, inodes []*Inode, fh fh.Fh) bool {
	return op.buftxn.CommitWait(true, false)
}

// Commit transaction, but don't write to stable storage
func CommitUnstable(op *FsTxn, inodes []*Inode, fh fh.Fh) bool {
	if len(inodes) > 1 {
		panic("commitUnstable")
	}
	return commitWait(op, inodes, false, false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func CommitFh(op *FsTxn, fh fh.Fh, inodes []*Inode) bool {
	putInodes(op)
	ok := op.buftxn.Flush()
	releaseInodes(op)
	return ok
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func Abort(op *FsTxn, inodes []*Inode) bool {
	return commitWait(op, inodes, true, true)
}
