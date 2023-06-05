#!/usr/bin/env python

__author__ = "Matthieu Rakotojaona Rainimangavelo, Marie Delavergne"
__credits__ = ["Matthieu Rakotojaona Rainimangavelo", "Marie Delavergne"]

import enoslib as en

en.init_logging()

time = "08:00:00"
site = "rennes"
cluster = "paravance"
nb_nodes = 3

cheops_location = "/tmp/cheops"
cheops_version = "master"



network = en.G5kNetworkConf(type="prod", roles=["my_network"], site=site)

conf = (
    en.G5kConf.from_settings(job_type="allow_classic_ssh", walltime=time, job_name="cheops")
    .add_network_conf(network)
    .add_machine(
        roles=["cheops"],
        cluster=cluster,
        nodes=nb_nodes,
        primary_network=network,
    )
    .finalize()
)


provider = en.G5k(conf)

roles, networks = provider.init()
roles = en.sync_info(roles, networks)



# Install and run kube
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.apt(
        update_cache=True,
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release"]
    )

    # docker
    p.apt_key(
        url="https://download.docker.com/linux/debian/gpg"
    )
    p.apt_repository(
        repo="deb https://download.docker.com/linux/debian {{ ansible_distribution_release }} stable",
        update_cache=True
    )
    p.apt(
        pkg=["docker-ce", "docker-ce-cli", "containerd.io"]
    )
    p.copy(
        dest="/etc/docker/daemon.json",
        content="""
        {
          "exec-opts": ["native.cgroupdriver=systemd"],
          "log-driver": "json-file",
          "log-opts": {
          "max-size": "100m"
          },
          "storage-driver": "overlay2"
        }
        """
    )
    p.systemd(
        daemon_reload=True,
        name="docker",
        state="started"
    )

    # kubernetes
    p.copy(
        dest="/etc/modules-load.d/k8s.conf",
        content="br_netfilter"
    )
    p.copy(
        dest="/etc/sysctl.d/k8s.conf",
        content="""
        net.bridge.bridge-nf-call-ip6tables = 1
        net.bridge.bridge-nf-call-iptables = 1
        """
    )
    p.shell(
        cmd="sysctl --system"
    )
    p.apt_key(
        url="https://packages.cloud.google.com/apt/doc/apt-key.gpg"
    )
    p.apt_repository(
        repo="deb https://apt.kubernetes.io/ kubernetes-xenial main",
        update_cache=True
    )
    p.apt(
        pkg=["kubelet=1.21.12-00", "kubeadm=1.21.12-00", "kubectl=1.21.12-00", "mount"]
    )
    p.command(
        cmd="apt-mark hold kubelet kubeadm kubectl"
    )
    p.command(
        cmd="swapoff -a"
    )



# Run kube
with en.actions(roles=roles["cheops"]) as p:
    p.shell(
        cmd="ss -ltnp | grep kubelet || kubeadm init --pod-network-cidr=10.244.0.0/16"
    )

# Apply config
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.file(
        path="{{ ansible_user_dir }}/.kube",
        state="directory"
    )
    p.copy(
        src="/etc/kubernetes/admin.conf",
        remote_src=True,
        dest="{{ ansible_user_dir }}/.kube/config"
    )
    # p.shell(
    #     cmd="kubectl apply -f https://github.com/coreos/flannel/raw/master/Documentation/kube-flannel.yml"
    # )


# Run ArangoDB

with en.actions(roles=roles["cheops"]) as p:
    p.apt(
        update_cache=True,
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release"]
    )
    p.apt_key(
        url="https://download.arangodb.com/arangodb38/DEBIAN/Release.key"
    )
    p.apt_repository(
        repo="deb https://download.arangodb.com/arangodb38/DEBIAN/ /",
        update_cache=True
    )
    p.apt(
        pkg=["arangodb3=3.8.0-1"]
    )
    p.copy(
        dest="/var/lib/arangodb3/LANGUAGE",
        content="""{"default":"en_US.UTF-8"}"""
    )
    p.systemd(
        daemon_reload=True,
        name="arangodb3",
        state="started"
    )
    p.wait_for(
        port=8529
    )
    p.shell(cmd="""(cat << EOF
db._createDatabase("cheops");
var users = require("@arangodb/users");
users.save("cheops@cheops", "lol");
users.grantDatabase("cheops@cheops", "cheops");
exit
EOF
)| arangosh --server.password ""
    """)

    # Check it is running
    p.uri(
        url="http://localhost:8529/_db/cheops/_api/database/current",
        user="cheops@cheops",
        password="lol",
        force_basic_auth=True
    )


# Install the correct Go version

with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.shell(cmd="curl -OL https://golang.org/dl/go1.17.linux-amd64.tar.gz || rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz")
    p.lineinfile(dest="/root/.bashrc",
                 line="export PATH=$PATH:/usr/local/go/bin",
                 insertafter="EOF",
                 regexp="export PATH=\$PATH:/usr/local/go/bin",
                 state="present")
    p.lineinfile(dest="/root/.bashrc",
                 line="export GOPATH=$HOME/go",
                 insertafter="EOF",
                 regexp="export GOPATH=\$HOME/go",
                 state="present")
    p.lineinfile(dest="/root/.bashrc",
                 line="export GOBIN=$GOPATH/bin",
                 insertafter="EOF",
                 regexp="export GOBIN=\$GOPATH/bin",
                 state="present")
    p.shell(cmd=". /root/.bashrc")
    p.shell(cmd="go get github.com/gorilla/mux || go get github.com/justinas/alice || go get github.com/arangodb/go-driver || go get github.com/arangodb/go-driver/http || go get github.com/segmentio/ksuid || go get github.com/rabbitmq/amqp091-go")


# Run RabbitMQ
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.shell(cmd="docker ps | grep rabbitmq:3.8-management || docker run -it -d --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.8-management")


# Get and run Cheops
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.git(
        repo="https://gitlab.inria.fr/discovery/cheops.git",
        dest=cheops_location,
        version=cheops_version
    )
    p.shell(
        cmd="go build",
        chdir=cheops_location
    )

# Allow to add sites to the conf file

for index, node in enumerate(roles["cheops"], start=1):
    site_name = "Site{i}".format(i=index)
    hostname = node.alias
    site_address = node.address
    site = "  - sitename: {name}\n    address: {address}".format(name=site_name, address=site_address)
    with en.actions(roles=roles["cheops"], gather_facts=True) as p:
        p.lineinfile(dest=cheops_location+"/config.yml",
                     line=site,
                     insertafter="knownsites:\n",
                     regexp=site,
                     state="present"
                     )
    with en.actions(roles=node, gather_facts=True) as p:
        p.lineinfile(dest=cheops_location+"/config.yml",
                     line=site,
                     insertafter="localsite:\n",
                     regexp=site,
                     state="present"
                     )
