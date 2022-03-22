package main

import (
	"flag"
	"fmt"
	"os"
	"path"
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

func client(duration time.Duration, p string) int {
	data := mkdata(uint64(100))
	start := time.Now()
	i := 0
	for {
		s := strconv.Itoa(i)
		smallfile(p+"/x"+s, data)
		i++
		if time.Since(start) >= duration {
			break
		}
	}
	return i
}

type config struct {
	dir      string // root for all clients
	duration time.Duration
}

func run(c config, nt int) (elapsed time.Duration, iters int) {
	start := time.Now()
	count := make(chan int)
	for i := 0; i < nt; i++ {
		p := path.Join(c.dir, "d"+strconv.Itoa(i))
		go func() {
			err := os.MkdirAll(p, 0700)
			if err != nil {
				panic(err)
			}
			count <- client(c.duration, p)
		}()
	}
	n := 0
	for i := 0; i < nt; i++ {
		n += <-count
	}
	return time.Since(start), n
}

func cleanup(c config, nt int) {
	for i := 0; i < nt; i++ {
		p := path.Join(c.dir, "d"+strconv.Itoa(i))
		os.Remove(p)
	}
}

func main() {
	var c config
	var start int
	var nthread int
	flag.StringVar(&c.dir, "dir", "/mnt/nfs", "root directory to run in")
	flag.DurationVar(&c.duration, "benchtime", 10*time.Second, "time to run each iteration for")
	flag.IntVar(&start, "start", 1, "number of threads to start at")
	flag.IntVar(&nthread, "threads", 1, "number of threads to run till")
	flag.Parse()
	if start < 1 {
		panic("invalid start")
	}

	// warmup (skip if running for very little time, for example when using a
	// duration of 0s to run just one iteration)
	if c.duration > 500*time.Millisecond {
		run(config{duration: 500 * time.Millisecond, dir: c.dir}, nthread)
	}

	for nt := start; nt <= nthread; nt++ {
		elapsed, count := run(c, nt)
		fmt.Printf("fs-smallfile: %v %0.4f file/sec\n", nt, float64(count)/elapsed.Seconds())
	}

	cleanup(c, nthread)
}
