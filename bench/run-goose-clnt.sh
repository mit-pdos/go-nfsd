#!/bin/sh

#
# Usage: ./run-goose-clnt.sh  go run ./cmd/clnt-smallfile/main.go
#

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# root of repo
cd $DIR/..

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
sleep 1
killall -0 goose-nfsd # make sure server is running
# taskset 0x3 $1 /mnt/nfs
echo "$@"
"$@"
killall goose-nfsd
