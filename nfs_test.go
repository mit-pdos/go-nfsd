package goose_nfs

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type NfsSuite struct {
	suite.Suite
	nfs *Nfs
}

func (suite *NfsSuite) SetupTest() {
	suite.nfs = MkNfs()
}

func (suite *NfsSuite) TestGetRoot() {
	nfs := suite.nfs
	args := &GETATTR3args{Object: MkRootFh3()}
	attr := &GETATTR3res{}

	res := nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	suite.Equal(attr.Resok.Obj_attributes.Ftype, NF3DIR)

	nfs.ShutdownNfs()
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

func (suite *NfsSuite) Lookup(name string) Nfs_fh3 {
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &LOOKUP3args{What: what}
	reply := &LOOKUP3res{}
	res := suite.nfs.Lookup(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
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

func (suite *NfsSuite) Setattr(fh Nfs_fh3, sz uint64) {
	size := Set_size3{Set_it: true, Size: Size3(sz)}
	attr := Sattr3{Size: size}
	args := &SETATTR3args{Object: fh, New_attributes: attr}
	reply := &SETATTR3res{}
	res := suite.nfs.SetAttr(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func (suite *NfsSuite) TestMakeFile() {
	suite.Create("x")
	fh := suite.Lookup("x")
	suite.Getattr(fh, 0)
	suite.Setattr(fh, 8192)
	suite.Getattr(fh, 8192)

	suite.nfs.ShutdownNfs()
}

func TestNfs(t *testing.T) {
	suite.Run(t, new(NfsSuite))
}
