#!/bin/sh

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &

# go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img -cpuprofile=nfsd.prof &

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR

./start-goose-nfs.sh || exit 1

function cleanup {
    ./stop-goose-nfs.sh
}
trap cleanup EXIT

# taskset 0x3 $1 /mnt/nfs
echo "run $@"
"$@"
