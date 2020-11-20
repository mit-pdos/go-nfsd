#!/bin/sh

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &

# go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img -cpuprofile=nfsd.prof &

./start-goose-nfs.sh

# taskset 0x3 $1 /mnt/nfs
echo "$@"
"$@"

./stop-goose-nfs.sh

