#!/bin/sh

#
# Usage:  ./run-goose-nfs.sh  go run ./cmd/fs-smallfile/main.go
#

# taskset 0xc go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &

go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img -cpuprofile=nfsd.prof &

# go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img &
sleep 1
sudo mount -t nfs -o vers=3 localhost:/ /mnt/nfs

# taskset 0x3 $1 /mnt/nfs
echo "$@"
"$@"

killall -s SIGINT goose-nfsd
sudo umount -f /mnt/nfs

