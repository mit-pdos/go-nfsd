package main

import (
	"fmt"
	"os"
	"time"
)

const (
	MB       uint64 = 1024 * 1024
	FILESIZE uint64 = 50 * MB
	WSIZE    uint64 = 16 * 4096
)

func makefile(name string, data []byte) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	for i := uint64(0); i < FILESIZE/WSIZE; i++ {
		_, err = f.Write(data)
		if err != nil {
			panic(err)
		}
	}
	err = f.Sync()
	if err != nil {
		panic(err)
	}
	err = f.Close()
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
	path := "/mnt/nfs/large"

	data := mkdata(WSIZE)
	start := time.Now()
	makefile(path, data)
	elapsed := time.Now().Sub(start)
	tput := float64(FILESIZE/MB) / elapsed.Seconds()
	fmt.Printf("fs-largefile: %v MB throughput %.2f MB/s\n", FILESIZE/MB, tput)
}
