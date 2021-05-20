package nfs

import (
	"testing"

	"github.com/mit-pdos/goose-nfsd/nfstypes"
	"github.com/stretchr/testify/assert"
)

func TestOpNames(t *testing.T) {
	assert := assert.New(t)

	// make sure nfsopNames list is sensible
	assert.Equal(NUM_NFS_OPS, len(nfsopNames))
	// first operation
	assert.Equal("NULL", nfsopNames[nfstypes.NFSPROC3_NULL])
	assert.Equal("FSINFO", nfsopNames[nfstypes.NFSPROC3_FSINFO])
	// the last operation
	assert.Equal("COMMIT", nfsopNames[nfstypes.NFSPROC3_COMMIT])
}
