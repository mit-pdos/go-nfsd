#!/bin/bash

set -e

#
# Usage: ./run-linux.sh go run ./cmd/fs-smallfile/main.go
#
# takes same flags as start-linux.sh but uses /dev/shm/nfs3.img as default disk
#

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# root of repo
cd "$DIR"/..

disk_file="/dev/shm/disk.img"
extra_args=()
while true; do
  case "$1" in
  -disk)
    shift
    disk_file="$1"
    shift
    ;;
  -*)
    extra_args+=("$1" "$2")
    shift
    shift
    ;;
  # stop gathering start-linux.sh flags as soon as a non-option is reached
  # remainder of command line is treated as command to run
  *)
    break
    ;;
  esac
done

./bench/start-linux.sh -disk "$disk_file" "${extra_args[@]}" || exit 1

function cleanup {
  ./bench/stop-linux.sh "$disk_file"
}
trap cleanup EXIT

echo "# Linux -disk $disk_file ${extra_args[*]}"
echo "run $*" 1>&2
"$@"
