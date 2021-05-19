package fstxn

// putInodes may free an inode so must be done before commit
func (op *FsTxn) preCommit() {
	op.Atxn.PreCommit()
}

func (op *FsTxn) postCommit() {
	op.releaseInodes()
	op.Atxn.PostCommit()
}

func (op *FsTxn) commitWait(wait bool) bool {
	op.preCommit()
	ok := op.Atxn.Op.CommitWait(wait)
	op.postCommit()
	return ok
}

func (op *FsTxn) Commit() bool {
	return op.commitWait(true)
}

// Commit data, but will also commit everything else, since we don't
// support log-by-pass writes.
func (op *FsTxn) CommitData() bool {
	return op.Atxn.Op.CommitWait(true)
}

// Commit transaction, but don't write to stable storage
func (op *FsTxn) CommitUnstable() bool {
	return op.commitWait(false)
}

// Flush log. We don't have to flush data from other file handles, but
// that is only an option if we do log-by-pass writes.
func (op *FsTxn) CommitFh() bool {
	op.preCommit()
	ok := op.Atxn.Op.Flush()
	op.postCommit()
	return ok
}

// An aborted transaction may free an inode, which results in dirty
// buffers that need to be written to log. So, call commit.
func (op *FsTxn) Abort() bool {
	op.releaseInodes()
	op.Atxn.PostAbort()
	return true
}
