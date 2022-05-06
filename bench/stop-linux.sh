#!/bin/sh

#
# Usage: ./stop-linux.sh [disk file]
#

native=false

while true; do
    case "$1" in
    -native=true)
        shift
        native=true
        ;;
    -native=false)
        shift
        native=false
        ;;
    *)
        break
        ;;
    esac
done

disk_file=/dev/shm/nfs3.img
if [ ! -z "$1" ]; then
    disk_file="$1"
fi

sudo umount -f /mnt/nfs

if [ "$native" = "false" ]; then
    sudo systemctl stop nfs-server.service
    sudo umount /srv/nfs/bench
fi

# do not attempt to remove block devices
if [ -f "$disk_file" ]; then
    rm -f "$disk_file"
fi
