#!/bin/bash

set -e

#
# Usage: ./start-linux.sh <disk file> [ext4_opts]
#
# set to /dev/shm/nfs3.img to use tmpfs, or a file to use the disk (through the host
# file system), or a block device to use a partition directly (NOTE: it will be
# overwritten; don't run as root)
#
# ext4_opts defaults to data=journal if not passed (use data=ordered for the
# default mode where metadata is journaled but not data)

# Requires /srv/nfs/bench to be set up for NFS export, otherwise you will get
#
# mount.nfs: access denied by server while mounting localhost:/srv/nfs/bench.
#
# 1. Create /srv/nfs/bench if it doesn't exist.
# 2. Edit /etc/exports and add the line:
# /srv/nfs/bench localhost(rw,sync,no_subtree_check)
# 3. Run
# sudo exportfs -arv
# to reload the export table
#

disk_file="$1"
ext4_opts="$2"

if [ -z "$ext4_opts" ]; then
  ext4_opts="data=journal"
fi

rm -f "$disk_file"
dd status=none if=/dev/zero of="$disk_file" bs=4K count=100000
mkfs.ext4 -q "$disk_file"
sync "$disk_file"
sudo mount -t ext4 -o "$ext4_opts" -o loop "$disk_file" /srv/nfs/bench
sudo systemctl start nfs-server.service
sudo mount -t nfs -o vers=3 localhost:/srv/nfs/bench /mnt/nfs
sudo chmod 777 /srv/nfs/bench
