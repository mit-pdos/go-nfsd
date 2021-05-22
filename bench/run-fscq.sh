#!/usr/bin/env bash
set -e

disk_file=/dev/shm/disk.img
fscq_path=""
mnt_path="/mnt/nfs"

while [[ "$#" -gt 0 ]]; do
    case "$1" in
    -disk)
        shift
        disk_file="$1"
        shift
        ;;
    -fscq)
        shift
        fscq_path="$1"
        shift
        ;;
    -mnt)
        shift
        mnt_path="$1"
        shift
        ;;
    -*)
        echo "Unexpected argument $1" 2>&1
        exit 1
        ;;
    *)
        break
        ;;
    esac
done

if [[ -z "$fscq_path" && -n "$FSCQ_PATH" ]]; then
    fscq_path="$FSCQ_PATH"
fi

if [ ! -d "$fscq_path" ]; then
    echo "Please set FSCQ_PATH or pass -fscq" 2>&1
    exit 1
fi

set -u

function cleanup {
    fusermount -u "$mnt_path"
    if [ -f "$disk_file" ]; then
        rm -f "$disk_file"
    fi
}
trap cleanup EXIT

"$fscq_path"/src/mkfs "$disk_file"
"$fscq_path"/src/fscq "$disk_file" \
    -o big_writes,atomic_o_trunc,use_ino,kernel_cache \
    "$mnt_path"
sleep 1
echo "# fscq -disk $disk_file"
echo "run $*" 1>&2
"$@"
