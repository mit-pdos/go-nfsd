#!/bin/sh

#
# Usage:  ./stop-goose-nfs.sh
#

killall -s SIGINT goose-nfsd
# sudo umount -f /mnt/nfs
