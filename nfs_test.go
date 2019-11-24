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

func (suite *NfsSuite) TestCreateLookup() {
	nfs := suite.nfs
	where := Diropargs3{Dir: MkRootFh3(), Name: "x"}
	how := Createhow3{}
	args := &CREATE3args{Where: where, How: how}
	attr := &CREATE3res{}
	res := nfs.Create(args, attr)
	suite.Require().Nil(res)
	suite.Equal(attr.Status, NFS3_OK)

	what := Diropargs3{Dir: MkRootFh3(), Name: "x"}
	arglook := &LOOKUP3args{What: what}
	lookres := &LOOKUP3res{}
	res1 := nfs.Lookup(arglook, lookres)
	suite.Require().Nil(res1)
	suite.Equal(lookres.Status, NFS3_OK)

	argsx := &GETATTR3args{Object: lookres.Resok.Object}
	attrx := &GETATTR3res{}
	res2 := nfs.GetAttr(argsx, attrx)
	suite.Require().Nil(res2)
	suite.Equal(attrx.Status, NFS3_OK)
	suite.Equal(attrx.Resok.Obj_attributes.Ftype, NF3REG)

	nfs.ShutdownNfs()
}

func TestNfs(t *testing.T) {
	suite.Run(t, new(NfsSuite))
}
