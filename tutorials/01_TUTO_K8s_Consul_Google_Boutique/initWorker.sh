#!/bin/bash
sudo apt-get update
sudo apt-get -y install docker.io
systemctl enable docker.service
sudo apt-get -y install curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt -y install software-properties-common
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
apt-get update
sudo apt-get -y install kubeadm kubelet kubectl
sudo apt-mark hold kubeadm kubelet kubectl
sudo swapoff -a
apt-get update
echo "#### COMPLETED ####"