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

// makefile directly reports throughput in MB/s
func makefile(name string, data []byte, size uint64) float64 {
	start := time.Now()
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	for i := uint64(0); i < size/WSIZE; i++ {
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
	elapsed := time.Now().Sub(start)
	return float64(size) / float64(MB) / elapsed.Seconds()
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
	sizeMB := flag.Uint64("file-size", 100, "file size (in MB)")
	deleteAfter := flag.Bool("delete", false, "delete files after running benchmark")
	warmup := flag.Bool("warmup", true, "run warmup first")
	flag.Parse()

	warmupFile := path.Join(*mnt, "large.warmup")
	file := path.Join(*mnt, "large")
	filesize := *sizeMB * MB
	warmupsize := 100 * MB
	if filesize < warmupsize {
		warmupsize = filesize
	}

	data := mkdata(WSIZE)

	if *warmup {
		tput := makefile(warmupFile, data, warmupsize)
		fmt.Printf("# warmup %d MB throughput %.2f MB/s\n", warmupsize/MB, tput)
	}

	tput := makefile(file, data, filesize)
	fmt.Printf("fs-largefile: %v MB throughput %.2f MB/s\n", filesize/MB, tput)

	if *deleteAfter {
		os.Remove(warmupFile)
		os.Remove(file)
	}
}
