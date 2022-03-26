package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// smallfile represents one iteration of this benchmark: it creates a file,
// write data to it, and deletes it.
func smallfile(dirFd int, name string, data []byte) {
	f, err := unix.Openat(dirFd, name, unix.O_CREAT|unix.O_RDWR, 0777)
	if err != nil {
		panic(err)
	}
	_, err = unix.Write(f, data)
	if err != nil {
		panic(err)
	}
	unix.Fsync(f)
	unix.Close(f)
	err = unix.Unlinkat(dirFd, name, 0)
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

type result struct {
	iters int
	times []time.Duration
}

func client(duration time.Duration, allTimes bool, rootDirFd int, path string) result {
	data := mkdata(uint64(100))
	var times []time.Duration
	if allTimes {
		times = make([]time.Duration, 0, int(duration.Seconds()*1000))
	}
	start := time.Now()
	i := 0
	var elapsed time.Duration
	for {
		s := strconv.Itoa(i)
		before := elapsed
		smallfile(rootDirFd, path+"/x"+s, data)
		i++
		elapsed = time.Since(start)
		if allTimes {
			times = append(times, (elapsed - before))
		}
		if elapsed >= duration {
			return result{iters: i, times: times}
		}
	}
}

type config struct {
	dir      string // root for all clients
	duration time.Duration
	allTimes bool // whether to record individual iteration timings
}

func run(c config, nt int) (elapsed time.Duration, iters int, times []time.Duration) {
	start := time.Now()
	count := make(chan result)
	rootDirFd, err := unix.Open(c.dir, unix.O_DIRECTORY, 0)
	if err != nil {
		panic(fmt.Errorf("could not open root directory fd: %v", err))
	}
	for i := 0; i < nt; i++ {
		i := i
		subdir := "d" + strconv.Itoa(i)
		p := path.Join(c.dir, subdir)
		go func() {
			err := os.MkdirAll(p, 0700)
			if err != nil {
				panic(err)
			}
			if err != nil {
				panic(err)
			}
			allTimes := c.allTimes && i == 0
			count <- client(c.duration, allTimes, rootDirFd, subdir)
		}()
	}
	for i := 0; i < nt; i++ {
		r := <-count
		iters += r.iters
		if r.times != nil {
			times = r.times
		}
	}
	elapsed = time.Since(start)
	return
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
	var timingFile string
	flag.StringVar(&c.dir, "dir", "/mnt/nfs", "root directory to run in")
	flag.DurationVar(&c.duration, "benchtime", 10*time.Second, "time to run each iteration for")
	flag.StringVar(&timingFile, "time-iters", "", "prefix for individual timing files")
	flag.IntVar(&start, "start", 1, "number of threads to start at")
	flag.IntVar(&nthread, "threads", 1, "number of threads to run till")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()
	if start < 1 {
		panic("invalid start")
	}

	// warmup (skip if running for very little time, for example when using a
	// duration of 0s to run just one iteration)
	if c.duration > 500*time.Millisecond {
		run(config{
			duration: 500 * time.Millisecond,
			dir:      c.dir,
			allTimes: false},
			nthread)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for nt := start; nt <= nthread; nt++ {
		if timingFile != "" {
			c.allTimes = true
		}

		elapsed, count, times := run(c, nt)
		fmt.Printf("fs-smallfile: %v %0.4f file/sec\n", nt,
			float64(count)/elapsed.Seconds())
		if len(times) > 0 {
			f, err := os.Create(fmt.Sprintf("%s-%d.txt", timingFile, nt))
			if err != nil {
				panic(fmt.Errorf("could not create timing file: %v", err))
			}
			for _, t := range times {
				fmt.Fprintf(f, "%f\n", t.Seconds())
			}
			f.Close()
		}
	}

	cleanup(c, nthread)
}
