#!/usr/bin/env bash

set -eu

blue=$(tput setaf 4)
reset=$(tput sgr0)

info() {
  echo -e "${blue}$1${reset}"
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
./run-goose-nfs.sh -disk ~/disk.img go run ./cmd/fs-smallfile -threads=$threads

echo
info "Linux smallfile scalability"
./run-linux.sh     -disk ~/disk.img go run ./cmd/fs-smallfile -threads=$threads
