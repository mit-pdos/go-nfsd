#!/bin/bash

set -e

#
# Usage: ./run-linux.sh go run ./cmd/fs-smallfile/main.go
#
# takes same flags as start-linux.sh but uses /dev/shm/nfs3.img as default disk
#

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR/..

fs="ext4"
mount_opts="data=journal"
disk_file=/dev/shm/nfs3.img

while true; do
  case "$1" in
    -disk)
      shift
      disk_file="$1"
      shift
      ;;
    -mount-opts)
      shift
      mount_opts="$1"
      shift
      ;;
    -fs)
      shift
      fs="$1"
      shift
      ;;
    *)
      break
      ;;
  esac
done

./bench/start-linux.sh -disk "$disk_file" -fs "$fs" -mount-opts "$mount_opts" || exit 1

function cleanup {
    ./bench/stop-linux.sh "$disk_file"
}
trap cleanup EXIT

echo "run $@" 1>&2
"$@"
