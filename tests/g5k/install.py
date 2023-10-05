#!/usr/bin/env python

import sys
import os

if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)

import enoslib as en
from datetime import datetime, timedelta

now = datetime.now()
if now.hour >= 19:
    max = datetime.today().replace(day=datetime.today().day +1 , hour=21, minute=0, second=0)
elif now.hour < 9:
    max = datetime.today().replace(hour=9, minute=0, second=0)
else:
    max = datetime.today().replace(hour=19, minute=0, second=0)

now_plus_1_hour = now + timedelta(minutes=58)

walltime = min((max - now), timedelta(minutes=55))
walltime = f"00:{walltime.seconds // 60}:00"

en.init_logging()

network = en.G5kNetworkConf(type="prod", roles=["my_network"], site="grenoble")
cluster = "dahu"

conf = (
    en.G5kConf.from_settings(job_type=[], walltime=walltime, job_name="cheops")
    .add_network_conf(network)
    .add_machine(
        roles=["cheops"],
        cluster=cluster,
        nodes=4,
        primary_network=network,
    )
    .finalize()
)

provider = en.G5k(conf)

roles, networks = provider.init()

en.sync_info(roles, networks)

with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.apt_key(url="https://couchdb.apache.org/repo/keys.asc")
    p.apt_repository(repo="deb https://apache.jfrog.io/artifactory/couchdb-deb/ {{ ansible_distribution_release }} main")
    p.shell(cmd="""
    COUCHDB_PASSWORD=password
    echo "couchdb couchdb/mode select standalone
    couchdb couchdb/mode seen true
    couchdb couchdb/bindaddress string 127.0.0.1
    couchdb couchdb/bindaddress seen true
    couchdb couchdb/cookie string elmo
    couchdb couchdb/cookie seen true
    couchdb couchdb/adminpass password ${COUCHDB_PASSWORD}
    couchdb couchdb/adminpass seen true
    couchdb couchdb/adminpass_again password ${COUCHDB_PASSWORD}
    couchdb couchdb/adminpass_again seen true" | debconf-set-selections
    """)
    p.apt(pkg="couchdb")
    p.lineinfile(
            path="/opt/couchdb/etc/local.ini",
            line="single_node=true",
            insertafter="\[couchdb\]"
    )    
    p.lineinfile(
            path="/opt/couchdb/etc/local.ini",
            line="bind_address = :: ",
            insertafter="\[chttpd\]"
    )
    p.systemd(
            name="couchdb",
            state="restarted"
    )

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
    p.shell(
        cmd="kubeadm init --pod-network-cidr=10.244.0.0/16"
    )    
    p.file(
        path="{{ ansible_user_dir }}/.kube",
        state="directory"
    )
    p.copy(
        src="/etc/kubernetes/admin.conf",
        remote_src=True,
        dest="{{ ansible_user_dir }}/.kube/config"
    )
    p.copy(
        src="{{ ansible_user_dir }}/.kube/config",
        remote_src=True,
        dest="{{ ansible_user_dir }}/.kube/config.proxified"
    )
    p.lineinfile(
        regexp=".*server:.*",
        line="    server: http://127.0.0.1:8079",
        path="{{ ansible_user_dir }}/.kube/config.proxified"
    )

with en.actions(roles=roles["cheops"], gather_facts=False) as p:
    p.apt(
        pkg="golang-1.19"
    )
