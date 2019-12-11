package goose_nfs

import "log"

const Debug = 5

func DPrintf(level int, format string, a ...interface{}) {
	if level <= Debug {
		log.Printf(format, a...)
	}
	return
}

func RoundUp(n uint64, sz uint64) uint64 {
	return (n + sz - 1) / sz
}
