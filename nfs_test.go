package goose_nfs

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	"testing"
)

type TestState struct {
	t   *testing.T
	nfs *Nfs
}

func (ts *TestState) Create(name string) {
	where := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	how := Createhow3{}
	args := &CREATE3args{Where: where, How: how}
	attr := &CREATE3res{}
	res := ts.nfs.Create(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3_OK)
}

func (ts *TestState) Lookup(name string, succeed bool) Nfs_fh3 {
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &LOOKUP3args{What: what}
	reply := &LOOKUP3res{}
	res := ts.nfs.Lookup(args, reply)
	assert.Nil(ts.t, res)
	if succeed {
		assert.Equal(ts.t, reply.Status, NFS3_OK)
	} else {
		assert.NotEqual(ts.t, reply.Status, NFS3_OK)
	}
	return reply.Resok.Object
}

func (ts *TestState) LookupDir(fh Nfs_fh3, name string, succeed bool) Nfs_fh3 {
	what := Diropargs3{Dir: fh, Name: Filename3(name)}
	args := &LOOKUP3args{What: what}
	reply := &LOOKUP3res{}
	res := ts.nfs.Lookup(args, reply)
	assert.Nil(ts.t, res)
	if succeed {
		assert.Equal(ts.t, reply.Status, NFS3_OK)
	} else {
		assert.NotEqual(ts.t, reply.Status, NFS3_OK)
	}
	return reply.Resok.Object
}

func (ts *TestState) Getattr(fh Nfs_fh3, sz uint64) {
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := ts.nfs.GetAttr(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3_OK)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Ftype, NF3REG)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Size, Size3(sz))
}

func (ts *TestState) GetattrDir(fh Nfs_fh3) {
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := ts.nfs.GetAttr(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3_OK)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Ftype, NF3DIR)
}

func (ts *TestState) GetattrFail(fh Nfs_fh3) {
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := ts.nfs.GetAttr(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3ERR_STALE)
}

func (ts *TestState) Setattr(fh Nfs_fh3, sz uint64) {
	size := Set_size3{Set_it: true, Size: Size3(sz)}
	attr := Sattr3{Size: size}
	args := &SETATTR3args{Object: fh, New_attributes: attr}
	reply := &SETATTR3res{}
	res := ts.nfs.SetAttr(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
}

func (ts *TestState) Write(fh Nfs_fh3, data []byte, how Stable_how) {
	args := &WRITE3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(len(data)),
		Stable: how,
		Data:   data}
	reply := &WRITE3res{}
	res := ts.nfs.Write(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
	assert.Equal(ts.t, reply.Resok.Count, Count3(len(data)))
}

func (ts *TestState) Read(fh Nfs_fh3, sz uint64) []byte {
	args := &READ3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(8192)}
	reply := &READ3res{}
	res := ts.nfs.Read(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
	assert.Equal(ts.t, reply.Resok.Count, Count3(sz))
	return reply.Resok.Data
}

func (ts *TestState) Remove(name string) {
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &REMOVE3args{
		Object: what,
	}
	reply := &REMOVE3res{}
	res := ts.nfs.Remove(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
}

func (ts *TestState) MkDir(name string) {
	where := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	sattr := Sattr3{}
	args := &MKDIR3args{Where: where, Attributes: sattr}
	attr := &MKDIR3res{}
	res := ts.nfs.MakeDir(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3_OK)
}

func (ts *TestState) Commit(fh Nfs_fh3, cnt uint64) {
	args := &COMMIT3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(cnt)}
	reply := &COMMIT3res{}
	res := ts.nfs.Commit(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
}

func (ts *TestState) Rename(from string, to string) {
	args := &RENAME3args{
		From: Diropargs3{Dir: MkRootFh3(), Name: Filename3(from)},
		To:   Diropargs3{Dir: MkRootFh3(), Name: Filename3(to)},
	}
	reply := &RENAME3res{}
	res := ts.nfs.Rename(args, reply)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, reply.Status, NFS3_OK)
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	l := uint64(len(data))
	for i := uint64(0); i < l; i++ {
		data[i] = byte(i % uint64(128))
	}
	return data
}

func (ts *TestState) readcheck(fh Nfs_fh3, data []byte) {
	d := ts.Read(fh, uint64(len(data)))
	assert.Equal(ts.t, len(data), len(d))
	for i := uint64(0); i < uint64(len(data)); i++ {
		assert.Equal(ts.t, data[i], d[i])
	}
}

func TestGetRoot(t *testing.T) {
	fmt.Printf("TestGetRoot\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	args := &GETATTR3args{Object: MkRootFh3()}
	attr := &GETATTR3res{}
	res := ts.nfs.GetAttr(args, attr)
	assert.Nil(ts.t, res)
	assert.Equal(ts.t, attr.Status, NFS3_OK)
	assert.Equal(ts.t, attr.Resok.Obj_attributes.Ftype, NF3DIR)
	ts.nfs.ShutdownNfs()
	fmt.Printf("TestGetRoot done\n")
}

// Grow file with setattr before writing
func TestFile(t *testing.T) {
	fmt.Printf("TestFile\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	sz := uint64(8192)
	ts.Create("x")
	fh := ts.Lookup("x", true)
	ts.Getattr(fh, 0)
	data := mkdata(sz)
	ts.Setattr(fh, sz)
	ts.Getattr(fh, sz)
	ts.Write(fh, data, FILE_SYNC)
	ts.readcheck(fh, data)
	ts.Remove("x")
	_ = ts.Lookup("x", false)
	ts.GetattrFail(fh)
	ts.nfs.ShutdownNfs()
	fmt.Printf("TestFile done\n")
}

// Grow file by writing
func TestFile1(t *testing.T) {
	fmt.Printf("TestFile1\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	sz := uint64(122)
	ts.Create("x")
	fh := ts.Lookup("x", true)
	data := mkdata(uint64(sz))
	ts.Write(fh, data, FILE_SYNC)
	ts.readcheck(fh, data)
	ts.nfs.ShutdownNfs()
	fmt.Printf("TestFile1 done\n")
}

func TestDir(t *testing.T) {
	fmt.Printf("TestDir\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	ts.MkDir("d")
	fh := ts.Lookup("d", true)
	ts.GetattrDir(fh)
	fhdot := ts.LookupDir(fh, ".", true)
	ts.GetattrDir(fhdot)
	// Rmdir("d")
	ts.nfs.ShutdownNfs()
	fmt.Printf("TestDir done\n")
}

// Many files
func TestManyFiles(t *testing.T) {
	fmt.Printf("TestManyFiles\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	for i := 0; i < 100; i++ {
		ts.Create("x")
		ts.Remove("x")
	}
	ts.nfs.ShutdownNfs()
	fmt.Printf("TestManyFiles done\n")
}

func TestRename(t *testing.T) {
	fmt.Printf("TestRename\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	ts.Create("x")
	ts.Rename("x", "y")
	_ = ts.Lookup("x", false)
	fh := ts.Lookup("y", true)
	ts.Getattr(fh, 0)

	ts.nfs.ShutdownNfs()
	fmt.Printf("TestRename done\n")
}

func TestUnstable(t *testing.T) {
	fmt.Printf("TestUnstable\n")
	ts := &TestState{t: t, nfs: MkNfs()}
	ts.Create("x")
	sz := uint64(4096)
	fh := ts.Lookup("x", true)
	data := mkdata(sz)
	ts.Write(fh, data, UNSTABLE)
	ts.Commit(fh, sz)
	ts.Write(fh, data, UNSTABLE)
	ts.Commit(fh, sz)

	ts.readcheck(fh, data)

	ts.nfs.ShutdownNfs()
	fmt.Printf("TestUnstable done\n")
}
