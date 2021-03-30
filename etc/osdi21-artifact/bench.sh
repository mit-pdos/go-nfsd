#!/bin/bash

# run various performance benchmarks

set -eo

cd "$GOOSE_NFSD_PATH"

./run-goose-nfs.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./run-goose-nfs.sh go run ./cmd/fs-largefile

./run-linux.sh go run ./cmd/fs-smallfile -start=10 -threads=10
./run-linux.sh go run ./cmd/fs-largefile
