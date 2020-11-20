#!/bin/sh

#
# Usage: ./stop-linux.sh
#

sudo umount -f /mnt/nfs
sudo systemctl stop nfs-server.service
sudo umount /srv/nfs/bench
