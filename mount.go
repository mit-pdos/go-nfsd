package goose_nfs

import (
	"log"
)

func (nfs *Nfs) MOUNTPROC3_NULL() {
	log.Printf("Null\n")
}

func (nfs *Nfs) MOUNTPROC3_MNT(args Dirpath3) Mountres3 {
	reply := new(Mountres3)
	log.Printf("Mount %v\n", args)
	reply.Fhs_status = MNT3_OK
	reply.Mountinfo.Fhandle = MkRootFh3().Data
	return *reply
}

func (nfs *Nfs) MOUNTPROC3_UMNT(args Dirpath3) {
	log.Printf("Unmount %v\n", args)
}

func (nfs *Nfs) MOUNTPROC3_UMNTALL() {
	log.Printf("Unmountall\n")
}

func (nfs *Nfs) MOUNTPROC3_DUMP() Mountopt3 {
	log.Printf("Dump\n")
	return Mountopt3{P: nil}
}

func (nfs *Nfs) MOUNTPROC3_EXPORT() Exportsopt3 {
	res := Exports3{
		Ex_dir:    "",
		Ex_groups: nil,
		Ex_next:   nil,
	}
	res.Ex_dir = "/"
	return Exportsopt3{P: &res}
}
