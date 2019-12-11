package goose_nfs

import "log"

const Debug = 0

func DPrintf(level int, format string, a ...interface{}) {
	if level <= Debug {
		log.Printf(format, a...)
	}
	return
}
