package goose_nfs

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"log"
)

type NfsSuite struct {
	suite.Suite
	nfs *Nfs
}

func (suite *NfsSuite) SetupTest() {
	suite.nfs = MkNfs()
}

func (suite *NfsSuite) TestGetRoot() {
	log.Printf("TestGetRoot\n")
	nfs := suite.nfs
	args := &GETATTR3args{Object: MkRootFh3()}
	attr := &GETATTR3res{}

	res := nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	suite.Equal(attr.Resok.Obj_attributes.Ftype, NF3DIR)

	nfs.ShutdownNfs()
	log.Printf("TestGetRoot done\n")
}

func (suite *NfsSuite) Create(name string) {
	where := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	how := Createhow3{}
	args := &CREATE3args{Where: where, How: how}
	attr := &CREATE3res{}
	res := suite.nfs.Create(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
}

func (suite *NfsSuite) Lookup(name string, succeed bool) Nfs_fh3 {
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &LOOKUP3args{What: what}
	reply := &LOOKUP3res{}
	res := suite.nfs.Lookup(args, reply)
	suite.Require().Nil(res)
	if succeed {
		suite.Equal(reply.Status, NFS3_OK)
	} else {
		suite.NotEqual(reply.Status, NFS3_OK)
	}
	return reply.Resok.Object
}

func (suite *NfsSuite) Getattr(fh Nfs_fh3, sz uint64) {
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := suite.nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	suite.Equal(attr.Resok.Obj_attributes.Ftype, NF3REG)
	suite.Equal(attr.Resok.Obj_attributes.Size, Size3(sz))
}

func (suite *NfsSuite) GetattrFail(fh Nfs_fh3) {
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := suite.nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3ERR_STALE)
}

func (suite *NfsSuite) Setattr(fh Nfs_fh3, sz uint64) {
	size := Set_size3{Set_it: true, Size: Size3(sz)}
	attr := Sattr3{Size: size}
	args := &SETATTR3args{Object: fh, New_attributes: attr}
	reply := &SETATTR3res{}
	res := suite.nfs.SetAttr(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func (suite *NfsSuite) Write(fh Nfs_fh3, data []byte, how Stable_how) {
	args := &WRITE3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(len(data)),
		Stable: how,
		Data:   data}
	reply := &WRITE3res{}
	res := suite.nfs.Write(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
	suite.Equal(reply.Resok.Count, Count3(len(data)))
}

func (suite *NfsSuite) Read(fh Nfs_fh3, sz uint64) []byte {
	args := &READ3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(8192)}
	reply := &READ3res{}
	res := suite.nfs.Read(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
	suite.Equal(reply.Resok.Count, Count3(sz))
	return reply.Resok.Data
}

func (suite *NfsSuite) Remove(name string) {
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &REMOVE3args{
		Object: what,
	}
	reply := &REMOVE3res{}
	res := suite.nfs.Remove(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func (suite *NfsSuite) Commit(fh Nfs_fh3, cnt uint64) {
	args := &COMMIT3args{
		File:   fh,
		Offset: Offset3(0),
		Count:  Count3(cnt)}
	reply := &COMMIT3res{}
	res := suite.nfs.Commit(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	l := uint64(len(data))
	for i := uint64(0); i < l; i++ {
		data[i] = byte(i % uint64(128))
	}
	return data
}

func (suite *NfsSuite) readcheck(fh Nfs_fh3, data []byte) {
	d := suite.Read(fh, uint64(len(data)))
	suite.Equal(len(data), len(d))
	for i := uint64(0); i < uint64(len(data)); i++ {
		suite.Equal(data[i], d[i])
	}
}

func (suite *NfsSuite) TestFile() {
	log.Printf("TestFile\n")
	sz := uint64(8192)
	suite.Create("x")
	fh := suite.Lookup("x", true)
	suite.Getattr(fh, 0)
	data := mkdata(sz)
	suite.Setattr(fh, sz)
	suite.Getattr(fh, sz)
	suite.Write(fh, data, FILE_SYNC)
	suite.readcheck(fh, data)
	suite.Remove("x")
	_ = suite.Lookup("x", false)
	suite.GetattrFail(fh)
	suite.nfs.ShutdownNfs()
	log.Printf("TestFile done\n")
}

func (suite *NfsSuite) TestFile1() {
	log.Printf("TestFile1\n")
	sz := uint64(122)
	suite.Create("x")
	fh := suite.Lookup("x", true)
	data := mkdata(uint64(sz))
	suite.Write(fh, data, FILE_SYNC)
	suite.readcheck(fh, data)
	suite.nfs.ShutdownNfs()
	log.Printf("TestFile1 done\n")
}

func (suite *NfsSuite) Rename(from string, to string) {
	args := &RENAME3args{
		From: Diropargs3{Dir: MkRootFh3(), Name: Filename3(from)},
		To:   Diropargs3{Dir: MkRootFh3(), Name: Filename3(to)},
	}
	reply := &RENAME3res{}
	res := suite.nfs.Rename(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func (suite *NfsSuite) TestRename() {
	log.Printf("TestRename\n")
	suite.Create("x")
	suite.Rename("x", "y")
	_ = suite.Lookup("x", false)
	fh := suite.Lookup("y", true)
	suite.Getattr(fh, 0)

	suite.nfs.ShutdownNfs()
	log.Printf("TestRename done\n")
}

func (suite *NfsSuite) TestUnstable() {
	log.Printf("TestUnstable\n")
	suite.Create("x")

	sz := uint64(4096)
	fh := suite.Lookup("x", true)
	data := mkdata(sz)
	suite.Write(fh, data, UNSTABLE)
	suite.Commit(fh, sz)
	suite.Write(fh, data, UNSTABLE)
	suite.Commit(fh, sz)

	suite.readcheck(fh, data)

	suite.nfs.ShutdownNfs()
	log.Printf("TestRename done\n")
}

func TestNfs(t *testing.T) {
	suite.Run(t, new(NfsSuite))
}
