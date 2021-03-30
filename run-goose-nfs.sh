#!/bin/bash

set -eu

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &

# go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img -cpuprofile=nfsd.prof &

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR

disk_file=/dev/shm/goose.img
if [ "$1" = "-disk" ]; then
    disk_file="$2"
    shift
    shift
fi

./start-goose-nfs.sh -disk "$disk_file" || exit 1

function cleanup {
    ./stop-goose-nfs.sh
}
trap cleanup EXIT

# taskset 0x3 $1 /mnt/nfs
echo "run $@" 1>&2
"$@"
