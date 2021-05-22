#!/usr/bin/env bash

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

if [ ! -d "$GO_NFSD_PATH" ]; then
    echo "\$GO_NFSD_PATH is unset" 1>&2
    exit 1
fi

if [ ! -d "$GO_JOURNAL_PATH" ]; then
    echo "\$GO_JOURNAL_PATH is unset" 1>&2
    exit 1
fi

help() {
    echo "Usage: $0 [-disk <disk file>] [threads]"
    echo "disk defaults to ~/disk.img (assuming the root file system is on an SSD)"
    echo "threads defaults to 10"
}

output_file="eval/data/scale-raw.txt"
disk_file="$HOME/disk.img"
while true; do
    case "$1" in
    -disk)
        shift
        disk_file="$1"
        shift
        ;;
    -o | --output)
        shift
        output_file="$1"
        shift
        ;;
    -help | --help)
        help
        exit 0
        ;;
    -*)
        error "unexpected flag $1"
        help
        exit 1
        ;;
    *)
        break
        ;;
    esac
done

threads=10
if [[ $# -gt 0 ]]; then
    threads="$1"
fi

cd "$GO_NFSD_PATH"

do_eval() {
    info "GoNFS smallfile scalability"
    echo "fs=gonfs"
    ./bench/run-go-nfsd.sh -disk "$disk_file" go run ./cmd/fs-smallfile -threads="$threads"

    echo 1>&2
    info "Linux smallfile scalability"
    echo "fs=linux"
    ./bench/run-linux.sh -disk "$disk_file" go run ./cmd/fs-smallfile -threads="$threads"

    echo 1>&2
    info "Serial GoNFS (holding locks)"

    # we change the local checkout of go-journal
    pushd "$GO_JOURNAL_PATH" >/dev/null
    git apply "$GO_NFSD_PATH/eval/serial.patch"
    popd >/dev/null
    # ... and then also point go-nfsd to the local version
    go mod edit -replace github.com/mit-pdos/go-journal="$GO_JOURNAL_PATH"

    echo "fs=serial-gonfs"
    ./bench/run-go-nfsd.sh -disk "$disk_file" go run ./cmd/fs-smallfile -start=1 -threads="$threads"

    go mod edit -dropreplace github.com/mit-pdos/go-journal
    pushd "$GO_JOURNAL_PATH"
    git restore wal/installer.go wal/logger.go wal/wal.go
    popd
}

if [ "$output_file" = "-" ]; then
    do_eval
else
    do_eval | tee "$output_file"
fi
