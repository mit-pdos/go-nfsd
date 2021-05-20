#!/bin/bash

# run various performance benchmarks

set -eu

blue=$(tput setaf 4)
red=$(tput setaf 1)
reset=$(tput sgr0)

info() {
    echo -e "${blue}$1${reset}" 1>&2
}
error() {
    echo -e "${red}$1${reset}" 1>&2
}

usage() {
    echo "Usage: $0 [-ssd <block device or file path>]" 1>&2
    echo "SSD benchmarks will be skipped if -ssd is not passed"
}

output_file="eval/data/bench-raw.txt"
ssd_file=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
    -ssd)
        shift
        ssd_file="$1"
        shift
        ;;
    -o | --output)
        shift
        output_file="$1"
        shift
        ;;
    -help | --help)
        usage
        exit 0
        ;;
    *)
        error "unexpected argument $1"
        usage
        exit 1
        ;;
    esac
done

if [ ! -d "$GO_NFSD_PATH" ]; then
    echo "GO_NFSD_PATH is unset" 1>&2
    exit 1
fi
if [ ! -d "$XV6_PATH" ]; then
    echo "XV6_PATH is unset" 1>&2
    exit 1
fi

cd "$GO_NFSD_PATH"

do_eval() {
    info "GoNFS"
    echo "fs=gonfs"
    ./bench/run-go-nfsd.sh -unstable=false -disk "" go run ./cmd/fs-smallfile -benchtime=20s
    ./bench/run-go-nfsd.sh -unstable=false -disk "" go run ./cmd/fs-largefile
    ./bench/run-go-nfsd.sh -unstable=false -disk "" ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs

    echo 1>&2
    info "Linux ext4 over NFS"
    echo "fs=linux"
    ./bench/run-linux.sh go run ./cmd/fs-smallfile -benchtime=20s
    ./bench/run-linux.sh go run ./cmd/fs-largefile
    ./bench/run-linux.sh ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs

    if [ -n "$ssd_file" ]; then
        echo 1>&2
        info "GoNFS (SSD)"
        echo "fs=gonfs-ssd"
        ./bench/run-go-nfsd.sh -unstable=false -disk "$ssd_file" go run ./cmd/fs-smallfile -benchtime=20s
        ./bench/run-go-nfsd.sh -unstable=false -disk "$ssd_file" go run ./cmd/fs-largefile
        ./bench/run-go-nfsd.sh -unstable=false -disk "$ssd_file" ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs

        echo "fs=gonfs-ssd-unstable"
        ./bench/run-go-nfsd.sh -unstable=true -disk "$ssd_file" go run ./cmd/fs-largefile
        echo "fs=gonfs-ssd-unstable-sync"
        ./bench/run-go-nfsd.sh -unstable=true -nfs-mount-opts "sync" -disk "$ssd_file" go run ./cmd/fs-largefile

        echo 1>&2
        info "Linux ext4 over NFS (SSD)"
        echo "fs=linux-ssd"
        ./bench/run-linux.sh -disk "$ssd_file" go run ./cmd/fs-smallfile -benchtime=20s
        ./bench/run-linux.sh -disk "$ssd_file" go run ./cmd/fs-largefile
        ./bench/run-linux.sh -disk "$ssd_file" ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs

        echo 1>&2
        echo "fs=linux-ssd-ordered"
        ./bench/run-linux.sh -disk "$ssd_file" -mount-opts "data=ordered" \
            go run ./cmd/fs-largefile
        echo "fs=linux-ssd-journal-sync"
        ./bench/run-linux.sh -disk "$ssd_file" -mount-opts "data=journal,sync" \
            go run ./cmd/fs-largefile
        echo "fs=linux-ssd-sync"
        ./bench/run-linux.sh -disk "$ssd_file" -mount-opts "data=journal" \
            -nfs-mount-opts "sync" \
            go run ./cmd/fs-largefile
    fi
}

if [ "$output_file" = "-" ]; then
    do_eval
else
    do_eval | tee "$output_file"
fi
