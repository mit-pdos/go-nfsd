#!/usr/bin/env bash
set -eu

# run the entire eval

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR"

./loc.py | tee data/lines-of-code.txt
./bench.sh | tee data/bench-raw.txt
./scale.sh 10 | tee data/scale-raw.txt
./plot.sh
