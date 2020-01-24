#!/bin/sh

#
# Usage: ./run-goose-clnt.sh  go run ./cmd/clnt-smallfile/main.go 
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
sleep 1
# taskset 0x3 $1 /mnt/nfs
echo "$@"
"$@"
killall goose-nfsd
