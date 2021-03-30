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
cd "$GOOSE_NFSD_PATH"

threads=10
if [[ $# -gt 0 ]]; then
    threads="$1"
fi

info "GoNFS smallfile scalability"
echo "fs=gonfs"
./run-goose-nfs.sh -disk ~/disk.img go run ./cmd/fs-smallfile -threads=$threads

echo 1>&2
info "Linux smallfile scalability"
echo "fs=linux"
./run-linux.sh     -disk ~/disk.img go run ./cmd/fs-smallfile -threads=$threads
