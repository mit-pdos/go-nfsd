package inode

import (
	"github.com/mit-pdos/goose-nfsd/fh"
	"github.com/mit-pdos/goose-nfsd/util"
)

// putInodes may free an inode so must be done before commit
func preCommit(op *FsTxn) {
	putInodes(op)
	op.commitAlloc()
}

func postCommit(op *FsTxn) {
	releaseInodes(op)
	op.commitFree()
}

func commitWait(op *FsTxn, wait bool, abort bool) bool {
	preCommit(op)
	ok := op.buftxn.CommitWait(wait, abort)
	postCommit(op)
	return ok
}

func Commit(op *FsTxn) bool {
	return commitWait(op, true, false)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func CommitData(op *FsTxn, fh fh.Fh) bool {
	return op.buftxn.CommitWait(true, false)
}

// Commit transaction, but don't write to stable storage
func CommitUnstable(op *FsTxn, fh fh.Fh) bool {
	return commitWait(op, false, false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func CommitFh(op *FsTxn, fh fh.Fh) bool {
	preCommit(op)
	ok := op.buftxn.Flush()
	postCommit(op)
	return ok
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func Abort(op *FsTxn) bool {
	putInodes(op)
	ok := op.buftxn.CommitWait(true, true)
	releaseInodes(op)
	util.DPrintf(1, "Abort: inum free %v alloc %v\n", op.freeInums, op.allocInums)
	util.DPrintf(1, "Abort: blk free %v alloc %v\n", op.freeBnums, op.allocBnums)
	return ok
}
