#!/bin/sh

#
# Usage: ./run-linux.sh go run ./cmd/fs-smallfile/main.go
#

./start-linux.sh
echo "run $@"
"$@"
./stop-linux.sh
