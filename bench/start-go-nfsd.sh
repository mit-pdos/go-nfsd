#!/bin/bash

set -eu

#
# Usage:  ./start-go-nfsd.sh <arguments>
#
# default disk is /dev/shm/goose.img but can be overriden by passing -disk again
#

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# root of repo
cd "$DIR"/..

nfs_mount_opts=""
extra_args=()
while [[ "$#" -gt 0 ]]; do
    case "$1" in
        -nfs-mount-opts)
            shift
            nfs_mount_opts="$1"
            shift
            ;;
        -*=*)
            extra_args+=("$1")
            shift
            ;;
        -*)
            extra_args+=("$1" "$2")
            shift
            shift
            ;;
    esac
done

go build ./cmd/go-nfsd
./go-nfsd -disk /dev/shm/goose.img "${extra_args[@]}" >nfs.out 2>&1 &
sleep 2
killall -0 go-nfsd       # make sure server is running
killall -SIGUSR1 go-nfsd # reset stats after recovery
sudo mount -t nfs -o "$nfs_mount_opts" localhost:/ /mnt/nfs
