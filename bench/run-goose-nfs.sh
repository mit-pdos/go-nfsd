#!/bin/bash

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &

# go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img -cpuprofile=nfsd.prof &

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR/..

disk_file=/dev/shm/goose.img
cpu_list=""
extra_args=()
while true; do
    case "$1" in
    -disk)
        shift
        disk_file="$1"
        shift
        ;;
    --cpu-list)
        shift
        cpu_list="$1"
        shift
        ;;
    # some argument in -foo=value syntax
    -*=*)
        extra_args+=("$1")
        shift
        ;;
    -*)
        extra_args+=("$1" "$2")
        shift
        shift
        ;;
    *)
        break
        ;;
    esac
done

set -eu

if [ -e "$disk_file" ]; then
    dd status=none if=/dev/zero of="$disk_file" bs=4K count=200000 conv=notrunc
    sync "$disk_file"
fi

if [ -z "$cpu_list" ]; then
    ./bench/start-goose-nfs.sh -disk "$disk_file" "${extra_args[@]}" || exit 1
else
    taskset --cpu-list "$cpu_list" ./bench/start-goose-nfs.sh -disk "$disk_file" "${extra_args[@]}" || exit 1
fi

function cleanup {
    ./bench/stop-goose-nfs.sh
    if [ -f "$disk_file" ]; then
        rm -f "$disk_file"
    fi
}
trap cleanup EXIT

# taskset 0x3 $1 /mnt/nfs
echo "# goose-nfsd -disk $disk_file ${extra_args[@]}" 1>&2
echo "run $@" 1>&2
"$@"
