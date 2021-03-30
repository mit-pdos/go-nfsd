#!/bin/bash

set -eu

#
# Usage: ./run-linux.sh go run ./cmd/fs-smallfile/main.go
#

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR

disk_file=/dev/shm/nfs3.img
if [ "$1" = "-disk" ]; then
    disk_file="$2"
    shift
    shift
fi

./start-linux.sh "$disk_file" || exit 1

function cleanup {
    ./stop-linux.sh
}
trap cleanup EXIT

echo "run $@" 1>&2
"$@"
