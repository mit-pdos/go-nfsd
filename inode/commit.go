package inode

import (
	"github.com/mit-pdos/goose-nfsd/util"
)

// putInodes may free an inode so must be done before commit
func (op *FsTxn) preCommit() {
	op.putInodes()
	op.commitAlloc()
}

func (op *FsTxn) postCommit() {
	op.releaseInodes()
	op.commitFree()
}

func (op *FsTxn) commitWait(wait bool, abort bool) bool {
	op.preCommit()
	ok := op.buftxn.CommitWait(wait, abort)
	op.postCommit()
	return ok
}

func (op *FsTxn) Commit() bool {
	return op.commitWait(true, false)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func (op *FsTxn) CommitData() bool {
	return op.buftxn.CommitWait(true, false)
}

// Commit transaction, but don't write to stable storage
func (op *FsTxn) CommitUnstable() bool {
	return op.commitWait(false, false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func (op *FsTxn) CommitFh() bool {
	op.preCommit()
	ok := op.buftxn.Flush()
	op.postCommit()
	return ok
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func (op *FsTxn) Abort() bool {
	op.putInodes()
	ok := op.buftxn.CommitWait(true, true)
	op.releaseInodes()
	util.DPrintf(1, "Abort: inum free %v alloc %v\n", op.freeInums, op.allocInums)
	util.DPrintf(1, "Abort: blk free %v alloc %v\n", op.freeBnums, op.allocBnums)
	return ok
}
