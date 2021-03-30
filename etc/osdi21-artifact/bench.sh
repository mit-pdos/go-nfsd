#!/bin/bash

# run various performance benchmarks

set -e

if [ ! -d "$GOOSE_NFSD_PATH" ]; then
    echo "GOOSE_NFSD_PATH is unset" 1>&2
    exit 1
fi
if [ ! -d "$XV6_PATH" ]; then
    echo "XV6_PATH is unset" 1>&2
    exit 1
fi

cd "$GOOSE_NFSD_PATH"

./run-goose-nfs.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./run-goose-nfs.sh go run ./cmd/fs-largefile
./run-goose-nfs.sh ./app-bench.sh "$XV6_PATH" /mnt/nfs

./run-linux.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./run-linux.sh go run ./cmd/fs-largefile
./run-linux.sh ./app-bench.sh "$XV6_PATH" /mnt/nfs
