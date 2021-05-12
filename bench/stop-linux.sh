#!/bin/sh

#
# Usage: ./stop-linux.sh [disk file]
#

disk_file=/dev/shm/nfs3.img
if [ ! -z "$1" ]; then
    disk_file="$1"
fi

sudo umount -f /mnt/nfs
sudo systemctl stop nfs-server.service
sudo umount /srv/nfs/bench
# do not attempt to remove block devices
if [ -f "$disk_file" ]; then
    rm -f "$disk_file"
fi
