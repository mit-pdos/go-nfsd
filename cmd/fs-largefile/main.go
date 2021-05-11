package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"
)

const (
	MB    uint64 = 1024 * 1024
	WSIZE        = 16 * 4096
)

var FILESIZE uint64

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
	mnt := flag.String("mnt", "/mnt/nfs", "directory to write files to")
	sizeMB := flag.Uint64("size", 100, "file size (in MB)")
	deleteAfter := flag.Bool("delete", false, "delete files after running benchmark")
	flag.Parse()

	warmupFile := path.Join(*mnt, "large.warmup")
	file := path.Join(*mnt, "large")
	FILESIZE = *sizeMB * MB

	data := mkdata(WSIZE)
	makefile(warmupFile, data)
	start := time.Now()
	makefile(file, data)
	elapsed := time.Now().Sub(start)
	tput := float64(FILESIZE/MB) / elapsed.Seconds()
	fmt.Printf("fs-largefile: %v MB throughput %.2f MB/s\n", FILESIZE/MB, tput)

	if *deleteAfter {
		os.Remove(warmupFile)
		os.Remove(file)
	}
}
