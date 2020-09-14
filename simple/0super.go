package simple

import (
	"github.com/mit-pdos/goose-nfsd/addr"
	"github.com/mit-pdos/goose-nfsd/common"
)

func block2addr(blkno common.Bnum) addr.Addr {
	return addr.MkAddr(blkno, 0)
}

func nInode() common.Inum {
	return common.Inum(common.INODEBLK)
}

func inum2Addr(inum common.Inum) addr.Addr {
	return addr.MkAddr(common.LOGSIZE, (uint64(inum) * common.INODESZ * 8))
}
