package kvs

import (
	"fmt"
	"log"
	"testing"

	"github.com/mit-pdos/goose-nfsd/addr"
)

const OBJSZ uint64 = 128

func mkdataval(b byte, sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = b
	}
	return data
}

func TestGetAndPuts(t *testing.T) {
	fmt.Printf("TestGetAndPuts\n")
	kvs := MkKVS()

	pairs := []KVPair{}
	addrs := []addr.Addr{}
	vals := [][]byte{}
	for i := 0; i < 10; i++ {
		addrs = append(addrs, addr.MkAddr(uint64(i), 0))
		vals = append(vals, mkdataval(byte(i), OBJSZ))
		pairs = append(pairs, KVPair{addrs[i], vals[i]})
	}

	ok := kvs.MultiPut(pairs)
	if !ok {
		log.Fatalf("Puts failed")
	}

	// ensure that get of a non-present key fails
	addrs = append(addrs, addr.MkAddr(10, 0))
	vals = append(vals, []byte{})
	for i := 0; i < 11; i++ {
		p := kvs.Get(addrs[i])
		for j := range p.Val {
			if p.Val[j] != vals[i][j] {
				log.Fatalf("Got %d, expected %d", p.Val, vals[i])
			}
		}
	}
}
