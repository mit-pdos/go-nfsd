package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func smallfile(name string, data []byte) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	_, err = f.Write(data)
	if err != nil {
		panic(err)
	}
	f.Close()
	err = os.Remove(name)
	if err != nil {
		panic(err)
	}
}

func mkdata(sz uint64) []byte {
	data := make([]byte, sz)
	for i := range data {
		data[i] = byte(i % 128)
	}
	return data
}

func main() {
	const N = 10 * time.Second
	path := "/mnt/nfs/x"

	start := time.Now()
	i := 0
	data := mkdata(uint64(100))
	for {
		s := strconv.Itoa(i)
		smallfile(path+s, data)
		i++
		t := time.Now()
		elapsed := t.Sub(start)
		if elapsed >= N {
			break
		}
	}
	fmt.Printf("fs-smallfile: %v file/sec\n", float64(i)/N.Seconds())
}
