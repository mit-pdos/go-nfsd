package goose_nfs

import "log"

const Debug uint64 = 0

func dPrintf(level uint64, format string, a ...interface{}) {
	if level <= Debug {
		log.Printf(format, a...)
	}
}

func roundUp(n uint64, sz uint64) uint64 {
	return (n + sz - 1) / sz
}

func min(n uint64, m uint64) uint64 {
	if n < m {
		return n
	} else {
		return m
	}
}
