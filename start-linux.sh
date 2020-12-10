#!/bin/sh

#
# Usage: ./start-linux.sh
#

IMG=/tmp/nfs3.img
# IMG=/home/kaashoek/tmp/nfs3.img
rm -f $IMG
dd if=/dev/zero of=$IMG bs=4K count=100000
mkfs -t ext3 $IMG
sudo mount -t ext3 -o data=journal -o loop $IMG /srv/nfs/bench
sudo systemctl start nfs-server.service
sudo mount -t nfs -o vers=3 localhost:/srv/nfs/bench /mnt/nfs
sudo chmod 777 /srv/nfs/bench
