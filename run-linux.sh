#!/bin/sh

#
# Usage: ./run-linux.sh go run ./cmd/fs-smallfile/main.go
#

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
# root of repo
cd $DIR

./start-linux.sh || exit 1

function cleanup {
    ./stop-linux.sh
}
trap cleanup EXIT

echo "run $@" 1>&2
"$@"
