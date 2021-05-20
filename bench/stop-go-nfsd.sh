#!/bin/sh

#
# Usage:  ./stop-go-nfsd.sh
#

killall -s SIGINT go-nfsd
sudo umount -f /mnt/nfs
