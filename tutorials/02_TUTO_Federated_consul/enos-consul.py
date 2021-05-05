import logging
from enoslib import *

PROVIDER_CONF = {
    "job_type": "allow_classic_ssh",
    "job_name": "Tdc1",
    "resources": {
        "machines": [
            {
                "roles": ["k8s-master","consul"],
                "cluster": "parasilo",
                "nodes": 1,
                "primary_network": "n1",
                "secondary_networks": [],
            },
            {
                "roles": ["k8s-worker"],
                "cluster": "parasilo",
                "nodes": 4,
                "primary_network": "n1",
                "secondary_networks": [],
            },
        ],
        "networks": [
            {"id": "n1", "type": "prod", "roles": ["my_network"], "site": "rennes"}
        ],
    },
}

# claim the resources
conf = G5kConf.from_dictionnary(PROVIDER_CONF)
provider = G5k(conf)

# Get actual resources
roles, networks = provider.init()
roles = sync_info(roles, networks)

docker = Docker(agent=get_hosts(roles=roles, pattern_hosts="all"), registry_opts={'type': 'external', 'ip': 'docker-cache.grid5000.fr', 'port': 80 })
docker.deploy()



# Install shared packages
with play_on(pattern_hosts="all", roles=roles) as yaml:
    yaml.apt(name=["curl", "software-properties-common"],
            update_cache=True)
   # yaml.systemd(name="docker.service", enabled=True)
    yaml.apt_key(url="https://packages.cloud.google.com/apt/doc/apt-key.gpg")
    yaml.apt_repository(repo="deb http://apt.kubernetes.io/ kubernetes-xenial main")
    yaml.apt(name=["kubeadm", "kubelet", "kubectl"], update_cache=True)
    yaml.dpkg_selections(name="kubeadm", selection="hold")
    yaml.dpkg_selections(name="kubelet", selection="hold")
    yaml.dpkg_selections(name="kubectl", selection="hold")
    yaml.shell("swapoff -a")
    yaml.file(path="/consul/data", state="directory")
    yaml.file(path='/consul', mode='777', recurse=True)

# Deploy k8s' master node
with play_on(pattern_hosts="k8s-master", roles=roles) as yaml:
    yaml.shell(command="kubeadm init --pod-network-cidr=10.244.0.0/16 && touch /tmp/kubeadm-init-done", creates="/tmp/kubeadm-init-done")
    yaml.shell(command="mkdir -p $HOME/.kube && touch /tmp/dir-done", creates="/tmp/dir-done")
    yaml.shell(command="cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && touch /tmp/cp-done", creates="/tmp/cp-done")
    yaml.shell(command="chown $(id -u):$(id -g) $HOME/.kube/config && touch /tmp/ch-done", creates="/tmp/ch-done")
    yaml.shell(command="kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml && touch /tmp/ap-done",
        creates="/tmp/ap-done")
    yaml.copy(src="pv.yaml", dest="/consul/data/pv.yaml")
    yaml.shell(command="kubectl apply -f /consul/data/pv.yaml && touch /tmp/pv-done", creates ="/tmp/pv-done")
    yaml.shell(command="curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3", creates="./get_helm.sh")
    yaml.file(path="get_helm.sh", mode='0700')
    yaml.shell(command="./get_helm.sh && touch /tmp/helm-done", creates="/tmp/helm-done")
    yaml.shell(command="helm repo add hashicorp https://helm.releases.hashicorp.com && touch /tmp/repo-done", creates="/tmp/repo-done")

# Get the command to join the cluster
output = run("kubeadm token create --print-join-command", hosts=roles['k8s-master'])

# Ask all worker nodes to join the cluster
with play_on(pattern_hosts="k8s-worker", roles=roles) as yaml:
    yaml.shell(command=output['ok'][roles['k8s-master'][0].alias]['stdout'] + "&& touch /tmp/join-done", creates="/tmp/join-done")

print("DONE")
