#!/bin/bash

# Usage: app-bench.sh xv6-repo path-to-fs
#
# path-to-fs should have the file system being tested mounted.
#
# Clones xv6-repo to path-to-fs/xv6, then compiles the kernel.
#

set -eu

if [ $# -ne 2 ]; then
    echo "$0 xv6-repo top-dir"
    exit 1
fi
xv6_repo="$1"
fs_dir="$2"

echo "=== app-bench $xv6_repo $fs_dir ===" 1>&2
cd "$fs_dir"

# run benchmark out of /tmp to avoid overhead of reading source repo
xv6_tmp="/tmp/xv6"
rm -rf "$xv6_tmp"
cp -r "$xv6_repo" "$xv6_tmp"

time_file="/tmp/time"

#echo "=== git clone ==="
/usr/bin/time -f "clone real %e" -o "$time_file" git clone --quiet "$xv6_tmp" xv6
clone_time="$(cut -d ' ' -f3 <"$time_file")"
cat "$time_file" 1>&2

#echo "=== compile xv6 ==="
cd xv6
/usr/bin/time -f "compile real %e user %U" -o "$time_file" make --quiet kernel
compile_time="$(cut -d ' ' -f3 <"$time_file")"
cat "$time_file" 1>&2
rm -f "$time_file"

total_time=$(awk "BEGIN{ print $clone_time + $compile_time }")
throughput=$(awk "BEGIN{ print 1.0 / $total_time }")
echo "total real $total_time s" 1>&2
echo "app-bench $throughput app/s"

rm -rf "$xv6_tmp"
