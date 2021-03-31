#!/bin/bash

# run various performance benchmarks

set -eu

blue=$(tput setaf 4)
reset=$(tput sgr0)

info() {
  echo -e "${blue}$1${reset}" 1>&2
}

if [ ! -d "$GOOSE_NFSD_PATH" ]; then
    echo "GOOSE_NFSD_PATH is unset" 1>&2
    exit 1
fi
if [ ! -d "$XV6_PATH" ]; then
    echo "XV6_PATH is unset" 1>&2
    exit 1
fi

cd "$GOOSE_NFSD_PATH"

info "GoNFS"
echo "fs=gonfs"
./bench/run-goose-nfs.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./bench/run-goose-nfs.sh go run ./cmd/fs-largefile
./bench/run-goose-nfs.sh ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs

echo 1>&2
info "Linux ext3 over NFS"
echo "fs=linux"
./bench/run-linux.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./bench/run-linux.sh go run ./cmd/fs-largefile
./bench/run-linux.sh ./bench/app-bench.sh "$XV6_PATH" /mnt/nfs
