#!/bin/sh

# Usage: app-bench.sh xv6-repo path-to-fs
#
# path-to-fs should have the file system being tested mounted.
#
# Clones xv6-repo to path-to-fs/xv6, then compiles the kernel.
#

set -e

if [ $# -ne 2 ]
 then
    echo "$0 xv6-repo top-dir"
    exit 1
fi

echo "=== app-bench $1 $2 ==="
cd "$2"

echo "=== git clone ==="
time -p git clone --quiet $1 xv6

echo "=== compile xv6 ==="
cd xv6
time -p make --quiet kernel
