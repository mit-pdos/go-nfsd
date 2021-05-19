#!/usr/bin/env bash

set -e

cd

sudo apt-get -y update
sudo apt-get -y upgrade
sudo apt-get -y install zsh

if [ ! -e ~/.oh-my-zsh ]; then
    wget https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh
    sh install.sh --unattended
    rm install.sh
fi
sudo chsh -s /usr/bin/zsh "$USER"

if [ ! -e goose-nfsd ]; then
    git clone https://github.com/mit-pdos/goose-nfsd
fi

git config --global pull.ff only
