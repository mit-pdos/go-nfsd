#!/bin/bash

set -eu

#
# Usage:  ./start-goose-nfs.sh <arguments>
#
# default disk is /dev/shm/goose.img but can be overriden by passing -disk again
#

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR/..

#go run ./cmd/goose-nfsd/ -disk ~/tmp/goose.img > nfs.out 2>&1 &
go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img "$@" > nfs.out 2>&1 &
sleep 2
killall -0 goose-nfsd # make sure server is running
sudo mount -t nfs -o vers=3 localhost:/ /mnt/nfs
