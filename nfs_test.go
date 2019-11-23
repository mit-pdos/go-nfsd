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

	res = nfs.GetAttr(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	suite.Equal(attr.Resok.Obj_attributes.Ftype, NF3DIR)

	nfs.ShutdownNfs()
}

func (suite *NfsSuite) TestCreate() {
	nfs := suite.nfs
	where := Diropargs3{Dir: MkRootFh3(), Name: "x"}
	how := Createhow3{}
	args := &CREATE3args{Where: where, How: how}
	attr := &CREATE3res{}
	res := nfs.Create(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)
	nfs.ShutdownNfs()
}

func TestNfs(t *testing.T) {
	suite.Run(t, new(NfsSuite))
}
