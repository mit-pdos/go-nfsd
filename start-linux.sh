#!/bin/sh

#
# Usage: ./start-linux.sh
#

mkdir -p /tmp/nfs
rm -f /tmp/nfs3.img
dd if=/dev/zero of=/tmp/nfs3.img bs=4K count=100000
mkfs -t ext3 /tmp/nfs3.img
sudo mount -t ext3 -o data=journal,sync -o loop /tmp/nfs3.img /srv/nfs/bench
sudo systemctl start nfs-server.service
sudo mount -t nfs -o vers=3 localhost:/srv/nfs/bench /mnt/nfs
sudo chmod 777 /srv/nfs/bench
