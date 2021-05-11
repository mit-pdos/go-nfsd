#!/usr/bin/env bash

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

threads=10
if [[ $# -gt 0 ]]; then
    threads="$1"
fi

disk_file="$HOME/disk.img"

cd "$GOOSE_NFSD_PATH"

info "GoNFS smallfile scalability"
echo "fs=gonfs"
./bench/run-goose-nfs.sh -disk "$disk_file" go run ./cmd/fs-smallfile -threads=$threads

echo 1>&2
info "Linux smallfile scalability"
echo "fs=linux"
./bench/run-linux.sh     -disk "$disk_file" go run ./cmd/fs-smallfile -threads=$threads

echo 1>&2
info "Serial GoNFS (holding locks)"
git apply osdi21-artifact/serial.patch
echo "fs=serial-gonfs"
./bench/run-goose-nfs.sh -disk "$disk_file" go run ./cmd/fs-smallfile -start=1 -threads=$threads
git restore wal/installer.go wal/logger.go
