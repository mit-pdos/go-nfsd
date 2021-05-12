package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/zeldovich/go-rpcgen/rfc1057"
	"github.com/zeldovich/go-rpcgen/rfc1813"
	"github.com/zeldovich/go-rpcgen/xdr"
)

var N time.Duration

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

func (c *nfsclnt) null() {
	var arg xdr.Void
	var res xdr.Void

	err := c.clnt.Call(rfc1813.NFSPROC3_NULL, c.cred, c.verf, &arg, &res)
	if err != nil {
		panic(err)
	}
}

func client(cred_unix rfc1057.Opaque_auth) (n int, elapsed time.Duration) {
	var cred_none rfc1057.Opaque_auth
	cred_none.Flavor = rfc1057.AUTH_NONE
	nfs := pmap_client("localhost", rfc1813.NFS_PROGRAM, rfc1813.NFS_V3)
	clnt := &nfsclnt{clnt: nfs, cred: cred_unix, verf: cred_none}
	start := time.Now()
	for {
		clnt.null()
		n++
		elapsed = time.Now().Sub(start)
		if elapsed >= N {
			return
		}
	}
}

func main() {
	flag.DurationVar(&N, "benchtime", 10*time.Second, "time to run each iteration for")
	flag.Parse()

	var err error

	var unix rfc1057.Auth_unix
	var cred_unix rfc1057.Opaque_auth
	cred_unix.Flavor = rfc1057.AUTH_UNIX
	cred_unix.Body, err = xdr.EncodeBuf(&unix)
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	n, elapsed := client(cred_unix)
	fmt.Printf("null-bench: %0.1f RPCs/s\n", float64(n)/elapsed.Seconds())
}
