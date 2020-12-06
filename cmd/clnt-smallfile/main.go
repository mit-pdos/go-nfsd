package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/rfc1813"
	"github.com/zeldovich/go-rpcgen/xdr"
)

const N = 10 * time.Second
const NTHREAD = 10

func pmap_client(host string, prog, vers uint32) *rfc1057.Client {
	var cred rfc1057.Opaque_auth
	cred.Flavor = rfc1057.AUTH_NONE

	pmapc, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, rfc1057.PMAP_PORT))
	if err != nil {
		panic(err)
	}
	defer pmapc.Close()
	pmap := rfc1057.MakeClient(pmapc, rfc1057.PMAP_PROG, rfc1057.PMAP_VERS)

	arg := rfc1057.Mapping{
		Prog: prog,
		Vers: vers,
		Prot: rfc1057.IPPROTO_TCP,
	}
	var res xdr.Uint32
	err = pmap.Call(rfc1057.PMAPPROC_GETPORT, cred, cred, &arg, &res)
	if err != nil {
		panic(err)
	}

	svcc, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, res))
	if err != nil {
		panic(err)
	}
	return rfc1057.MakeClient(svcc, prog, vers)
}

type nfsclnt struct {
	clnt *rfc1057.Client
	cred rfc1057.Opaque_auth
	verf rfc1057.Opaque_auth
}

func (c *nfsclnt) getattr(fh rfc1813.Nfs_fh3) *rfc1813.GETATTR3res {
	var arg rfc1813.GETATTR3args
	var res rfc1813.GETATTR3res

	arg.Object = fh
	err := c.clnt.Call(rfc1813.NFSPROC3_GETATTR, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func (c *nfsclnt) lookup(fh rfc1813.Nfs_fh3, name string) *rfc1813.LOOKUP3res {
	var res rfc1813.LOOKUP3res

	what := rfc1813.Diropargs3{Dir: fh, Name: rfc1813.Filename3(name)}
	arg := rfc1813.LOOKUP3args{What: what}
	err := c.clnt.Call(rfc1813.NFSPROC3_LOOKUP, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func (c *nfsclnt) create(fh rfc1813.Nfs_fh3, name string) *rfc1813.CREATE3res {
	var res rfc1813.CREATE3res

	where := rfc1813.Diropargs3{Dir: fh, Name: rfc1813.Filename3(name)}
	how := rfc1813.Createhow3{}
	arg := rfc1813.CREATE3args{Where: where, How: how}
	err := c.clnt.Call(rfc1813.NFSPROC3_CREATE, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func (c *nfsclnt) remove(fh rfc1813.Nfs_fh3, name string) *rfc1813.REMOVE3res {
	var res rfc1813.REMOVE3res
	what := rfc1813.Diropargs3{Dir: fh, Name: rfc1813.Filename3(name)}
	arg := rfc1813.REMOVE3args{
		Object: what,
	}

	err := c.clnt.Call(rfc1813.NFSPROC3_REMOVE, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func (c *nfsclnt) write(fh rfc1813.Nfs_fh3, off uint64, data []byte, how rfc1813.Stable_how) *rfc1813.WRITE3res {
	var res rfc1813.WRITE3res

	arg := rfc1813.WRITE3args{
		File:   fh,
		Offset: rfc1813.Offset3(off),
		Count:  rfc1813.Count3(len(data)),
		Stable: how,
		Data:   data}
	err := c.clnt.Call(rfc1813.NFSPROC3_WRITE, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func smallfile(clnt *nfsclnt, dirfh rfc1813.Nfs_fh3, name string, data []byte) {
	reply := clnt.lookup(dirfh, name)
	if reply.Status == rfc1813.NFS3_OK {
		panic("smallfile")
	}
	clnt.create(dirfh, name)
	reply = clnt.lookup(dirfh, name)
	if reply.Status != rfc1813.NFS3_OK {
		panic("smallfile")
	}
	attr := clnt.getattr(reply.Resok.Object)
	if attr.Status != rfc1813.NFS3_OK {
		panic("SmallFile")
	}
	clnt.write(reply.Resok.Object, 0, data, rfc1813.FILE_SYNC)
	attr = clnt.getattr(reply.Resok.Object)
	if attr.Status != rfc1813.NFS3_OK {
		panic("smallfile")
	}
	res := clnt.remove(dirfh, name)
	if res.Status != rfc1813.NFS3_OK {
		panic("smallfile")
	}
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func client(i int, root_fh rfc1813.Nfs_fh3, cred_unix rfc1057.Opaque_auth, cred_none rfc1057.Opaque_auth, count chan int) {
	nfs := pmap_client("localhost", rfc1813.NFS_PROGRAM, rfc1813.NFS_V3)
	clnt := &nfsclnt{clnt: nfs, cred: cred_unix, verf: cred_none}
	data := mkdata(uint64(100))
	n := 0
	start := time.Now()
	for true {
		s := strconv.Itoa(i)
		smallfile(clnt, root_fh, "x"+s, data)
		n++
		t := time.Now()
		elapsed := t.Sub(start)
		if elapsed >= N {
			count <- n
			break
		}
	}
}

func pclient(root_fh rfc1813.Nfs_fh3, cred_unix rfc1057.Opaque_auth, cred_none rfc1057.Opaque_auth) {
	for t := 1; t <= NTHREAD; t++ {
		count := make(chan int)
		for i := 1; i <= t; i++ {
			go client(i, root_fh, cred_unix, cred_none, count)
		}
		n := 0
		for i := 0; i < t; i++ {
			c := <-count
			n += c
		}
		fmt.Printf("clnt-smallfile: %v %v file/s\n", t, float64(n)/N.Seconds())
	}
}

func main() {
	var err error

	var unix rfc1057.Auth_unix
	var cred_unix rfc1057.Opaque_auth
	cred_unix.Flavor = rfc1057.AUTH_UNIX
	cred_unix.Body, err = xdr.EncodeBuf(&unix)
	if err != nil {
		panic(err)
	}

	var cred_none rfc1057.Opaque_auth
	cred_none.Flavor = rfc1057.AUTH_NONE

	mnt := pmap_client("localhost", rfc1813.MOUNT_PROGRAM, rfc1813.MOUNT_V3)

	arg := rfc1813.Dirpath3("/")
	var res rfc1813.Mountres3
	err = mnt.Call(rfc1813.MOUNTPROC3_MNT, cred_none, cred_none, &arg, &res)
	if err != nil {
		panic(err)
	}

	if res.Fhs_status != rfc1813.MNT3_OK {
		panic(fmt.Sprintf("mount status %d", res.Fhs_status))
	}

	var root_fh rfc1813.Nfs_fh3
	root_fh.Data = res.Mountinfo.Fhandle

	for _, flavor := range res.Mountinfo.Auth_flavors {
		fmt.Printf("flavor %d\n", flavor)
	}

	pclient(root_fh, cred_unix, cred_none)
}
