#!/bin/sh

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
sudo mount -t nfs -o vers=3 localhost:/ /mnt/nfs 
# taskset 0x3 $1 /mnt/nfs
echo "$@"
"$@"
sudo umount -f /mnt/nfs
killall goose-nfsd
