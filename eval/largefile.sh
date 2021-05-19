#!/bin/bash

# for experimenting with the many modes of largefile

set -e

blue=$(tput setaf 4 || printf "")
reset=$(tput sgr0 || printf "")

info() {
    echo -e "${blue}$1${reset}"
}

info "goose-nfsd"
./bench/run-goose-nfs.sh -disk "/dev/shm/disk.img" -unstable=true ./fs-largefile
./bench/run-goose-nfs.sh -disk "/dev/shm/disk.img" -unstable=false ./fs-largefile
./bench/run-goose-nfs.sh -disk "" -unstable=true ./fs-largefile
./bench/run-goose-nfs.sh -disk "" -unstable=false ./fs-largefile

echo
info "Linux (ext4)"
./bench/run-linux.sh -fs ext4 -disk "/dev/shm/disk.img" -mount-opts "data=journal" ./fs-largefile
./bench/run-linux.sh -fs ext4 -disk "/dev/shm/disk.img" -mount-opts "data=ordered" ./fs-largefile
./bench/run-linux.sh -fs ext4 -disk "/dev/shm/disk.img" -mount-opts "sync,data=journal" ./fs-largefile
./bench/run-linux.sh -fs ext4 -disk "/dev/shm/disk.img" -mount-opts "sync,data=ordered" ./fs-largefile

echo
info "Linux (btrfs)"
./bench/run-linux.sh -fs btrfs -disk "/dev/shm/disk.img" -mount-opts "" ./fs-largefile
./bench/run-linux.sh -fs btrfs -disk "/dev/shm/disk.img" -mount-opts "flushoncommit" ./fs-largefile
