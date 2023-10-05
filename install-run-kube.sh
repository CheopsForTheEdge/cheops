#!/usr/bin/env sh

apt update && apt install -y curl gnupg apt-transport-https ca-certificates lsb-release

curl https://download.docker.com/linux/debian/gpg | gpg --dearmor > /usr/share/keyrings/docker-keyring.gpg
source /etc/os-release
echo "deb https://download.docker.com/linux/debian ${VERSION_CODENAME} stable" > /etc/apt/sources.list.d/docker.list
apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io

# Docker
cat <<EOF > /etc/docker/daemon.json
{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "log-driver": "json-file",
  "log-opts": {
  "max-size": "100m"
  },
  "storage-driver": "overlay2"
}
EOF

systemctl daemon-reload
systemctl restart docker

# Kube
echo "br_netfilter" > /etc/modules.load.d/k8s.conf
cat <<EOF > /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF

sysctl --system

curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor > /usr/share/keyrings/kubernetes-keyring.gpg
source /etc/os-release
echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
apt-get update && apt-get install -y kubelet=1.21.12-00 kubeadm=1.21.12-00 kubectl=1.21.12-00 mount
apt-mark hold kubelet kubeadm kubectl
swapoff -a
kubeadm init --pod-network-cidr=10.244.0.0/16

mkdir -p ~/.kube
cp /etc/kubernetes/admin.conf ~/.kube/config
