#!/usr/bin/env bash
set -eu

# run the entire eval

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR"

./loc.py | tee data/lines-of-code.txt
./bench.sh -ssd ~/disk.img
./scale.sh 12
./plot.sh
