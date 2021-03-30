#!/bin/sh

#
# Usage: ./start-linux.sh
#

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

IMG=/tmp/nfs3.img
# IMG=/home/kaashoek/tmp/nfs3.img
rm -f $IMG
dd status=none if=/dev/zero of=$IMG bs=4K count=100000
mkfs.ext3 -q $IMG
sudo mount -t ext3 -o data=journal -o loop $IMG /srv/nfs/bench
sudo systemctl start nfs-server.service
sudo mount -t nfs -o vers=3 localhost:/srv/nfs/bench /mnt/nfs
sudo chmod 777 /srv/nfs/bench
