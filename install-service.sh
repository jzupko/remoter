#!/bin/bash

# see also: http://stackoverflow.com/questions/59895/can-a-bash-script-tell-what-directory-its-stored-in
SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Can fail
sudo systemctl stop remoter
sudo systemctl disable remoter
sudo rm -f /etc/systemd/system/remoter.service

# Remaining can't fail
set -e

go build
sudo cp ${SCRIPTDIR}/remoter.service /etc/systemd/system/remoter.service
sudo chmod 644 /etc/systemd/system/remoter.service
sudo chown root:root /etc/systemd/system/remoter.service
sudo systemctl start remoter
sudo systemctl enable remoter
