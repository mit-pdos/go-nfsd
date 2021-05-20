#!/usr/bin/env bash
set -eu

cd "$GO_NFSD_PATH"
./bench/run-go-nfsd.sh $LTP_PATH/testcases/kernel/fs/fsstress/fsstress \
    -l 200 -n 100 -p 4 -d /mnt/nfs
./bench/run-go-nfsd.sh $LTP_PATH/testcases/kernel/fs/fsx-linux/fsx-linux \
    -N 100000 /mnt/nfs/x
