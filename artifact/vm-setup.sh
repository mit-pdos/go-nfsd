#!/usr/bin/env bash

set -eu

install_ocaml=true
install_coq=true
while [[ "$#" -gt 0 ]]; do
    case "$1" in
    -no-ocaml)
        install_ocaml=false
        shift
        ;;
    -ocaml)
        install_ocaml=true
        shift
        ;;
    -no-coq)
        install_coq=false
        shift
        ;;
    -coq)
        install_ocaml=true
        install_coq=true
        shift
        ;;
    *)
        echo "Unexpected argument $1" 1>&2
        exit 1
        ;;
    esac
done

cd

# Install really basic dependencies

sudo apt-get update
sudo apt-get install -y git python3-pip wget unzip psmisc sudo time

# Get source code

## assumes https://github.com/mit-pdos/go-nfsd has already been cloned to
## ~/go-nfsd (since this is the easiest way to run this script)
ln -s ~/go-nfsd/artifact ~/artifact

git clone \
    --branch osdi21 \
    --recurse-submodules \
    https://github.com/mit-pdos/perennial

mkdir ~/code
cd ~/code
git clone https://github.com/mit-pdos/go-journal &
git clone https://github.com/mit-pdos/xv6-public &
git clone https://github.com/tchajed/marshal &
git clone https://github.com/tchajed/goose &
wait
git clone --depth=1 https://github.com/linux-test-project/ltp
cd

cat >>~/.profile <<EOF
export GO_NFSD_PATH=$HOME/go-nfsd
export PERENNIAL_PATH=$HOME/perennial

export GO_JOURNAL_PATH=$HOME/code/go-journal
export MARSHAL_PATH=$HOME/code/marshal
export XV6_PATH=$HOME/code/xv6-public
export GOOSE_PATH=$HOME/code/goose
export LTP_PATH=$HOME/code/ltp
EOF

echo -e "\nsource ~/.profile" >>~/.zshrc

# Set up NFS client and server

sudo apt-get install -y rpcbind nfs-common nfs-kernel-server
sudo mkdir -p /srv/nfs/bench
sudo chown "$USER:$USER" /srv/nfs/bench
sudo mkdir -p /mnt/nfs
sudo chown "$USER:$USER" /mnt/nfs
echo "/srv/nfs/bench localhost(rw,sync,no_subtree_check,fsid=0)" | sudo tee -a /etc/exports

## for simplicity we enable these services so they are automatically started,
## but they can instead be started manually on each boot
sudo systemctl enable rpcbind
sudo systemctl enable rpc-statd
sudo systemctl disable nfs-server
# can't run go-nfsd and Linux NFS server at the same time
sudo systemctl stop nfs-server
sudo systemctl start rpcbind
sudo systemctl start rpc-statd

# Set up Linux file-system tests

sudo apt-get install -y autoconf m4 automake pkg-config
cd ~/code/ltp
make autotools
./configure
make -C testcases/kernel/fs/fsstress
make -C testcases/kernel/fs/fsx-linux
cd

# Install Python dependencies

pip3 install argparse pandas

# gnuplot (for creating graphs)

sudo apt-get install -y gnuplot-nox

# Install Go and Go dependencies

GO_FILE=go1.16.4.linux-amd64.tar.gz
wget https://golang.org/dl/$GO_FILE
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf $GO_FILE
rm $GO_FILE
# shellcheck disable=2016
echo 'export PATH=$HOME/go/bin:/usr/local/go/bin:$PATH' >>~/.profile
export PATH=/usr/local/go/bin:$PATH

go install github.com/tchajed/goose/cmd/goose@latest
# these are required in $GOPATH for goose to compile go-nfsd
export GOPATH=$HOME/go
export GO111MODULE=off
go get github.com/tchajed/goose/...
go get github.com/mit-pdos/go-journal/...
go get github.com/mit-pdos/go-nfsd/...
export GO111MODULE=on

cd ~/go-nfsd
# fetch dependencies
go build ./cmd/go-nfsd && rm go-nfsd
cd

# Install Coq

# opam dependencies
sudo apt-get install -y m4 bubblewrap
# coq dependencies
sudo apt-get install -y libgmp-dev

# use binary installer for opam since it has fewer dependencies than Ubuntu
# package
wget https://raw.githubusercontent.com/ocaml/opam/master/shell/install.sh
# echo is to answer question about where to install opam
echo "" | sh install.sh --no-backup
rm install.sh

opam init --auto-setup --bare
if [ "$install_ocaml" = true ]; then
    opam switch create 4.11.0+flambda

    # shellcheck disable=2046
    eval $(opam env)

    if [ "$install_coq" = "true" ]; then
        opam install -y -j4 coq.8.13.2
    fi
fi

sudo apt clean
opam clean
