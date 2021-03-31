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

# make sure code is compiled in case it takes longer than 2s to build
go build ./cmd/goose-nfsd && rm -f goose-nfsd
go run ./cmd/goose-nfsd/ -disk /dev/shm/goose.img "$@" > nfs.out 2>&1 &
sleep 2
killall -0 goose-nfsd # make sure server is running
sudo mount -t nfs -o vers=3 localhost:/ /mnt/nfs
