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

ssd_file=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -ssd)
            shift
            ssd_file="$1"
            shift
            ;;
        *)
            error "unexpected argument $1"
            usage
            exit 1
            ;;
    esac
done

if [ ! -d "$GOOSE_NFSD_PATH" ]; then
    echo "GOOSE_NFSD_PATH is unset" 1>&2
    exit 1
fi

cd "$GOOSE_NFSD_PATH"

info "GoNFS"
./bench/run-goose-nfs.sh -stats true -unstable=false -disk "" go run ./cmd/fs-smallfile -benchtime=20s
cat nfs.out > osdi21-artifact/data/gonfs-latencies.txt
cat osdi21-artifact/data/gonfs-latencies.txt

echo 1>&2
info "Linux ext4 over NFS"
sudo bpftrace ./osdi21-artifact/nfsdist.bt > osdi21-artifact/data/linux-latencies.txt &
./bench/run-linux.sh go run ./cmd/fs-smallfile -benchtime=20s
sudo killall bpftrace
sleep 1
cat osdi21-artifact/data/linux-latencies.txt

if [ -n "$ssd_file" ]; then
    echo 1>&2
    info "GoNFS (SSD)"
    echo "fs=gonfs-ssd"
    ./bench/run-goose-nfs.sh -stats true -unstable=false -disk "$ssd_file" go run ./cmd/fs-smallfile -benchtime=20s
    cat nfs.out > osdi21-artifact/data/gonfs-disk-latencies.txt
    cat osdi21-artifact/data/gonfs-disk-latencies.txt

    echo 1>&2
    info "Linux ext4 over NFS (SSD)"
    sudo bpftrace ./osdi21-artifact/nfsdist.bt > osdi21-artifact/data/linux-disk-latencies.txt &
    ./bench/run-linux.sh -disk "$ssd_file" go run ./cmd/fs-smallfile -benchtime=20s
    sudo killall bpftrace
    sleep 1
    cat osdi21-artifact/data/linux-disk-latencies.txt
fi

