#!/usr/bin/env bash

set -eu

cd

# Install really basic dependencies

sudo apt-get update
sudo apt-get install -y git python3-pip wget

# Get source code

## assumes https://github.com/mit-pdos/goose-nfsd has already been cloned to
## ~/goose-nfsd (since this is the easiest way to run this script)
git clone --recurse-submodules https://github.com/mit-pdos/perennial

mkdir ~/code
cd ~/code
git clone https://github.com/mit-pdos/xv6-public
git clone https://github.com/tchajed/marshal
git clone https://github.com/tchajed/goose
cd

cat >> ~/.profile <<EOF
export GOOSE_NFSD_PATH=$HOME/goose-nfsd
export PERENNIAL_PATH=$HOME/perennial
export MARSHAL_PATH=$HOME/code/marshal
export XV6_PATH=$HOME/code/xv6-public
export GOOSE_PATH=$HOME/code/goose
EOF

# Set up NFS client and server

sudo apt-get install -y rpcbind nfs-common nfs-server
sudo mkdir -p /srv/nfs/bench
sudo chown $USER:$USER /srv/nfs/bench
sudo mkdir -p /mnt/nfs
sudo chown $USER:$USER /mnt/nfs
echo "/srv/nfs/bench localhost(rw,sync,no_subtree_check)" | sudo tee -a /etc/exports

## for simplicity we enable these services so they are automatically started,
## but they can instead be started manually on each boot
sudo systemctl enable rpcbind
sudo systemctl enable rpc-statd
sudo systemctl disable nfs-server
# can't run goose-nfsd and Linux NFS server at the same time
sudo systemctl stop nfs-server

# Install Python dependencies

pip3 install argparse pandas

# gnuplot (for creating graphs)

sudo apt-get install -y gnuplot-nox

# Install Go and Go dependencies

GO_FILE=go1.16.2.linux-amd64.tar.gz
wget https://golang.org/dl/$GO_FILE
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf $GO_FILE
rm $GO_FILE
echo 'export PATH=$HOME/go/bin:/usr/local/go/bin:$PATH' >> ~/.profile
export PATH=/usr/local/go/bin:$PATH

go install github.com/tchajed/goose/cmd/goose@latest
# these are required in $GOPATH for goose to compile goose-nfsd
export GOPATH=$HOME/go
export GO111MODULE=off
go get github.com/tchajed/goose/...
go get github.com/mit-pdos/goose-nfsd/...

# Install Coq

sudo apt-get install -y libgmp-dev
# use binary installer for opam since it has fewer dependencies than Ubuntu
# package
sh <(curl -sL https://raw.githubusercontent.com/ocaml/opam/master/shell/install.sh)
opam init --auto-setup --bare
opam switch create 4.11.0+flambda
eval $(opam env)
opam install -y -j4 coq.8.13.1
