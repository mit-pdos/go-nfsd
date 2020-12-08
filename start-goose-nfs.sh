#!/bin/sh

#
# Usage:  ./start-goose-nfs.sh
#

dd if=/dev/zero of=~/tmp/goose.img bs=4K count=100000
go run ./cmd/goose-nfsd/ -disk ~/tmp/goose.img > nfs.out 2>&1 &

#go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img > nfs.out 2>&1 &

sleep 2

#sudo mount -t nfs -o vers=3 localhost:/ /mnt/nfs
