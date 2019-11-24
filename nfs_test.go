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
	nfs := suite.nfs
	where := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	how := Createhow3{}
	args := &CREATE3args{Where: where, How: how}
	attr := &CREATE3res{}
	res := nfs.Create(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
}

func (suite *NfsSuite) Lookup(name string) Nfs_fh3 {
	nfs := suite.nfs
	what := Diropargs3{Dir: MkRootFh3(), Name: Filename3(name)}
	args := &LOOKUP3args{What: what}
	reply := &LOOKUP3res{}
	res := nfs.Lookup(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
	return reply.Resok.Object
}

func (suite *NfsSuite) Getattr(fh Nfs_fh3) {
	nfs := suite.nfs
	args := &GETATTR3args{Object: fh}
	attr := &GETATTR3res{}
	res := nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	suite.Equal(attr.Resok.Obj_attributes.Ftype, NF3REG)
}

func (suite *NfsSuite) Setattr(fh Nfs_fh3, sz uint64) {
	nfs := suite.nfs
	size := Set_size3{Set_it: true, Size: Size3(sz)}
	attr := Sattr3{Size: size}
	args := &SETATTR3args{Object: fh, New_attributes: attr}
	reply := &SETATTR3res{}
	res := nfs.SetAttr(args, reply)
	suite.Require().Nil(res)
	suite.Equal(reply.Status, NFS3_OK)
}

func (suite *NfsSuite) TestMakeFile() {
	nfs := suite.nfs
	suite.Create("x")
	fh := suite.Lookup("x")
	suite.Getattr(fh)
	suite.Setattr(fh, 8192)
	nfs.ShutdownNfs()
}

func TestNfs(t *testing.T) {
	suite.Run(t, new(NfsSuite))
}
