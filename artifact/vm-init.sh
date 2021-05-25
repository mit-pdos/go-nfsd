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
sed -i 's/ZSH_THEME=.*/ZSH_THEME="dst"/' ~/.zshrc
rm -f ~/.bashrc

sudo passwd -d "$USER"
sudo sed -e 's/#PermitEmptyPasswords no/PermitEmptyPasswords yes/' -i /etc/ssh/sshd_config
sudo systemctl restart sshd

if [ ! -e go-nfsd ]; then
    git clone https://github.com/mit-pdos/go-nfsd
fi

git config --global pull.ff only
