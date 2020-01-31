package fs

import (
	"fmt"
	"testing"

	"github.com/mit-pdos/goose-nfsd/wal"
	"github.com/tchajed/goose/machine/disk"
	"gotest.tools/assert"
)

func mkData(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func checkData(t *testing.T, read, expected []byte) {
	assert.Equal(t, len(read), len(expected))
	for i := uint64(0); i < uint64(len(read)); i++ {
		assert.Equal(t, read[i], expected[i])
	}
}

func checkBlk(t *testing.T, fs *FsSuper, blkno uint64, expected []byte) {
	d := fs.Disk.Read(blkno + uint64(fs.DataStart()))
	checkData(t, d, expected)
}

func TestRecoverNone(t *testing.T) {
	fmt.Printf("TestRecoverNone\n")
	fs := MkFsSuper(100*1000, nil)

	b := wal.MkBlockData(0, mkData(disk.BlockSize))

	l := wal.MkLog(fs.Disk)
	l.Shutdown()

	_, ok := l.MemAppend([]wal.BlockData{b})
	assert.Equal(t, ok, true)

	checkBlk(t, fs, 0, make([]byte, disk.BlockSize))

	l.Recover()

	checkBlk(t, fs, 0, make([]byte, disk.BlockSize))
}

func TestRecoverSimple(t *testing.T) {
	fmt.Printf("TestRecoverSimple\n")
	fs := MkFsSuper(100*1000, nil)
	d := mkData(disk.BlockSize)

	b := wal.MkBlockData(fs.DataStart(), d)

	l := wal.MkLog(fs.Disk)

	txn, ok := l.MemAppend([]wal.BlockData{b})
	assert.Equal(t, ok, true)
	l.LogAppendWait(txn)

	l.Shutdown()

	l.Recover()

	checkBlk(t, fs, 0, d)
}
