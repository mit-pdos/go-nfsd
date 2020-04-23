package kvs

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mit-pdos/goose-nfsd/common"
	"github.com/tchajed/goose/machine/disk"
)

const DISKSZ = 10000

func mkdataval(b byte, sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = b
	}
	return data
}

func TestGetAndPuts(t *testing.T) {
	fmt.Printf("TestGetAndPuts\n")

	os.Remove(DISKNAME)
	d, err := disk.NewFileDisk(DISKNAME, DISKSZ)
	if err != nil {
		panic(fmt.Errorf("could not create file disk: %v", err))
	}
	kvs := MkKVS(d, DISKSZ)

	pairs := []KVPair{}
	keys := []uint64{}
	vals := [][]byte{}
	for i := 0; i < 10; i++ {
		keys = append(keys, common.LOGSIZE+uint64(i))
		vals = append(vals, mkdataval(byte(i), disk.BlockSize))
		pairs = append(pairs, KVPair{keys[i], vals[i]})
	}

	ok := kvs.MultiPut(pairs)
	if !ok {
		log.Fatalf("Puts failed")
	}

	for i := 0; i < 10; i++ {
		p, ok := kvs.Get(keys[i])
		if !ok {
			log.Fatalf("Get failed?")
		}
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
	err = os.Remove(DISKNAME)
	if err != nil {
		panic(err)
	}
}
