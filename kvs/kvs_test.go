package kvs

import (
	"fmt"
	"log"
	"testing"

	"github.com/tchajed/goose/machine/disk"
)

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
	keys := []uint64{}
	vals := [][]byte{}
	for i := 0; i < 10; i++ {
		keys = append(keys, uint64(i))
		vals = append(vals, mkdataval(byte(i), disk.BlockSize))
		pairs = append(pairs, KVPair{keys[i], vals[i]})
	}

	ok := kvs.MultiPut(pairs)
	if !ok {
		log.Fatalf("Puts failed")
	}

	for i := 0; i < 10; i++ {
		p := kvs.Get(keys[i])
		for j := range p.Val {
			if p.Val[j] != vals[i][j] {
				log.Fatalf("%d: Got %d, expected %d", i, p.Val[j], vals[i][j])
			}
		}
	}
	/*keys = append(keys, 12)
	if kvs.Get(keys[10]) != nil {
		log.Fatalf("Returned nonpresent key")
	}*/
	kvs.Delete()
}
