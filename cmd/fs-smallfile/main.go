package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

var N time.Duration
var NTHREAD int

func smallfile(name string, data []byte) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	_, err = f.Write(data)
	if err != nil {
		panic(err)
	}
	f.Sync()
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

func client(p string) int {
	data := mkdata(uint64(100))
	start := time.Now()
	i := 0
	for {
		s := strconv.Itoa(i)
		smallfile(p+"/x"+s, data)
		i++
		t := time.Now()
		elapsed := t.Sub(start)
		if elapsed >= N {
			break
		}
	}
	return i
}

func run(nt int) {
	path := "/mnt/nfs/"
	count := make(chan int)
	for i := 0; i < nt; i++ {
		d := "d" + strconv.Itoa(i)
		go func(d string) {
			err := os.MkdirAll(path+"/"+d+"/", 0700)
			if err != nil {
				panic(err)
			}
			count <- client(path + d)
		}(d)

	}
	n := 0
	for i := 0; i < nt; i++ {
		n += <-count
	}
	fmt.Printf("fs-smallfile: %v %v file/sec\n", nt, float64(n)/N.Seconds())
}

func main() {
	var start int
	flag.DurationVar(&N, "benchtime", 10*time.Second, "time to run each iteration for")
	flag.IntVar(&start, "start", 1, "number of threads to start at")
	flag.IntVar(&NTHREAD, "threads", 1, "number of threads to run till")
	flag.Parse()
	if start < 1 {
		panic("invalid start")
	}

	for i := start; i <= NTHREAD; i++ {
		run(i)
	}
}
