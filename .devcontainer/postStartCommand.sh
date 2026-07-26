#!/bin/sh

echo "postStartCommand.sh"
echo "-------------------"

sudo apt-get update
sudo apt-get upgrade -y

go version
kubectl version --client --output=yaml | head -2

echo "Done"
