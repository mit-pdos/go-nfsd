#!/bin/bash

set -e

#
# Usage: ./start-linux.sh [-disk path] [-mount-opts opts] [-fs fs]
#
# set to /dev/shm/nfs3.img to use tmpfs, or a file to use the disk (through the host
# file system), or a block device to use a partition directly (NOTE: it will be
# overwritten; don't run as root)
#
# fs defaults to ext4
#
# opts defaults to data=journal if fs is ext3 or ext4 if not passed (use
# data=ordered for the default mode where metadata is journaled but not data)

# Requires /srv/nfs/bench to be set up for NFS export, otherwise you will get
#
# mount.nfs: access denied by server while mounting localhost:/srv/nfs/bench.
#
# 1. Create /srv/nfs/bench if it doesn't exist.
# 2. Edit /etc/exports and add the line:
# /srv/nfs/bench localhost(rw,sync,no_subtree_check,fsid=0)
# 3. Run
# sudo exportfs -arv
# to reload the export table
#

fs="ext4"
disk_file=""
mount_opts=""
nfs_mount_opts=""
size_mb=400

while true; do
    case "$1" in
    -disk)
        shift
        disk_file="$1"
        shift
        ;;
    -mount-opts)
        shift
        mount_opts="$1"
        shift
        ;;
    -nfs-mount-opts)
        shift
        nfs_mount_opts="$1"
        shift
        ;;
    -fs)
        shift
        fs="$1"
        shift
        ;;
    -size)
        shift
        size_mb="$1"
        shift
        ;;
    *)
        break
        ;;
    esac
done

set -u

if [[ "$fs" == "ext4" ]] || [[ "$fs" = "ext3" ]]; then
    if [ -z "$mount_opts" ]; then
        mount_opts="data=journal"
    fi
fi

if [ -z "$disk_file" ]; then
    echo "-disk not provided" >&2
    exit 1
fi

conv_arg=()
if [ ! -b "$disk_file" ]; then
    conv_arg+=("conv=notrunc")
fi

_nfs_mount="vers=3,wsize=65536,rsize=65536"
if [ -n "$nfs_mount_opts" ]; then
    _nfs_mount="${_nfs_mount},$nfs_mount_opts"
fi

# count is in units of 4KB blocks
dd status=none if=/dev/zero of="$disk_file" bs=4K "${conv_arg[@]}" count=$((size_mb * 1024 / 4))
mkfs."$fs" -q "$disk_file"
sync "$disk_file"
sudo mount -t "$fs" -o "$mount_opts" -o loop "$disk_file" /srv/nfs/bench
sudo systemctl start nfs-server.service
sudo mount -t nfs -o "${_nfs_mount}" localhost:/srv/nfs/bench /mnt/nfs
sudo chmod 777 /srv/nfs/bench
