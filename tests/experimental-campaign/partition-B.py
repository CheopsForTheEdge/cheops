#!/usr/bin/env python

__author__ = "Matthieu Rakotojaona Rainimangavelo, Marie Delavergne"
__credits__ = ["Matthieu Rakotojaona Rainimangavelo", "Marie Delavergne"]


import re
import requests
import json
import yaml
import time
import datetime
import os
import tarfile


import enoslib as en

en.init_logging()

experience_time = "03:00:00"
site = "rennes"
cluster = "paravance"
experience_name = "partition-A"
nb_nodes = 5
nb_replicas = 4

cheops_location = "/tmp/cheops"
cheops_version = "raft-stable"




network = en.G5kNetworkConf(type="prod", roles=["my_network"], site=site)

conf = (
    en.G5kConf.from_settings(job_type="allow_classic_ssh",
                             walltime=experience_time,
                             job_name=experience_name)
    .add_network_conf(network)
    .add_machine(
        roles=["cheops"],
        cluster=cluster,
        nodes=nb_nodes-1,
        primary_network=network,
    )
    .add_machine(
        roles=["cheops", "faulty"],
        cluster=cluster,
        nodes=1,
        primary_network=network,
    )
    .finalize()
)


provider = en.G5k(conf)

roles, networks = provider.init()
roles = en.sync_info(roles, networks)





# Docker
registry_opts = dict(type="external", ip="docker-cache.grid5000.fr", port=80)

# https://stackoverflow.com/a/1774043
with open("docker_login.yml", "r") as stream:
    try:
        dockerlogin = yaml.safe_load(stream)
        dockername = dockerlogin['username']
        dockertoken = dockerlogin['token']
    except yaml.YAMLError as exc:
        print(exc)

d = en.Docker(
    agent=roles["cheops"],
    bind_var_docker="/tmp/docker",
    registry_opts=registry_opts,
     credentials=dict(login=dockername, password=dockertoken),

)
d.deploy()


with en.actions(roles=roles["cheops"], gather_facts=False) as p:
    p.file(
        task_name="Make a directory for results",
        path="/tmp/results/",
        state="directory"
    )


# Install and run kube
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.apt(
        update_cache=True,
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release"]
    )

    # kubernetes
    p.copy(
        task_name="Copy kubernetes conf",
        dest="/etc/modules-load.d/k8s.conf",
        content="br_netfilter"
    )
    p.copy(
        task_name="Copy kubernetes systemcl conf with content",
        dest="/etc/sysctl.d/k8s.conf",
        content="""
        net.bridge.bridge-nf-call-ip6tables = 1
        net.bridge.bridge-nf-call-iptables = 1
        """
    )
    p.shell(
        task_name="Sysctl --system",
        cmd="sysctl --system"
    )
    p.apt_key(
        task_name="Adding google apt key",
        url="https://packages.cloud.google.com/apt/doc/apt-key.gpg"
    )
    p.apt_repository(
        task_name="Get kubernetes",
        repo="deb https://apt.kubernetes.io/ kubernetes-xenial main",
        update_cache=True
    )
    p.apt(
        task_name="Get kubelet, kubeadm, kubectl, mount",
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
        task_name="Kubeadm init",
        cmd="ss -ltnp | grep kubelet || kubeadm init --pod-network-cidr=10.244.0.0/16"
    )

# Do the rest of the kube config
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.file(
        task_name="Make a new kube directory",
        path="{{ ansible_user_dir }}/.kube",
        state="directory"
    )
    p.copy(
        task_name="Copy kube admin config to the user directory",
        src="/etc/kubernetes/admin.conf",
        remote_src=True,
        dest="{{ ansible_user_dir }}/.kube/config"
    )
    p.copy(
        task_name="Copy kubeconfig to the correct file (proxified)",
        src="{{ ansible_user_dir }}/.kube/config",
        remote_src=True,
        dest="{{ ansible_user_dir }}/.kube/config.proxified"
    )
    p.lineinfile(
        task_name="Add port  to kube config",
        regexp=".*server:.*",
        line="    server: http://127.0.0.1:8079",
        path="{{ ansible_user_dir }}/.kube/config.proxified"
    )
    p.service(
        task_name="Restart kubelet",
        name="kubelet",
        state="restarted"
    )
    p.shell(
        task_name="Correct 'pending' status",
        cmd="kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml"
    )

# Run ArangoDB
with en.actions(roles=roles["cheops"]) as p:
    p.apt(
        task_name="Getting prerequisites for ArangoDB",
        update_cache=True,
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release"]
    )
    p.apt_key(
        task_name="Getting ArangoDB apt key",
        url="https://download.arangodb.com/arangodb38/DEBIAN/Release.key"
    )
    p.apt_repository(
        task_name="Adding ArangoDB repo",
        repo="deb https://download.arangodb.com/arangodb38/DEBIAN/ /",
        update_cache=True
    )
    p.apt(
        task_name="Getting ArangoDB",
        pkg=["arangodb3=3.8.0-1"]
    )
    p.copy(
        task_name="Configure ArangoDB langague",
        dest="/var/lib/arangodb3/LANGUAGE",
        content="""{"default":"en_US.UTF-8"}"""
    )
    p.systemd(
        task_name="Starting ArangoDB",
        daemon_reload=True,
        name="arangodb3",
        state="started"
    )
    p.wait_for(
        task_name="Wait for ArangoDB",
        port=8529
    )
    p.shell(
        task_name="Configure ArangoDB",
        cmd="""(cat << EOF
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
        task_name="Check arangoDB",
        url="http://localhost:8529/_db/cheops/_api/database/current",
        user="cheops@cheops",
        password="lol",
        force_basic_auth=True
    )



# Run RabbitMQ
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.shell(cmd="docker ps | grep rabbitmq:3.8-management || docker run -it -d --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.8-management")



# Get and run Cheops
with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.apt(
        task_name="Get golang",
        pkg="golang-1.19"
    )
    p.git(
        task_name="Download cheops",
        repo="https://gitlab.inria.fr/discovery/cheops.git",
        dest=cheops_location,
        version=cheops_version,
        update=False
    )
    p.shell(
        task_name="Build cheops",
        cmd="go build",
        chdir=cheops_location,
        environment={"PATH": "/usr/lib/go-1.19/bin: {{ ansible_env.PATH }}"}
    )
    p.shell(
        task_name="Remove cheops log",
        cmd=f"rm {cheops_location}/cheops.log; killall cheops.com 2> /dev/null",
        ignore_errors=True
    )
    p.shell(
        task_name="Remove raft tmp files",
        cmd="rm -r /tmp/raft/*",
        ignore_errors=True
    )
    p.shell(
        task_name="Config cheops",
        cmd="""
        MYIP={{{{ ansible_default_ipv4.address }}}} \
        MYFQDN={{{{ ansible_fqdn }}}} \
        STATE_DIR=/tmp/raft \
        MODE=raft \
        {loc}/cheops.com 2> /tmp/results/cheops.log
        """.format(loc=cheops_location),
        chdir=cheops_location,
        background=True
    )
    p.wait_for(
        task_name="Wait for raft to be available",
        port=7071,
        timeout=100
    )


# Preparing to test Raft
time.sleep(5)

makeid = lambda fqdn: re.sub(r'{cluster}-(\d*).{site}.grid5000.fr'.format(cluster=cluster,
                                                                          site=site), r'\1', fqdn)
peers = [{"Address": r.address + ":7070", "ID": int(makeid(r.address))} for r in roles["cheops"]]
payload = json.dumps({"GroupID": 0, "Peers": peers})

with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.uri(
        task_name="Create groups",
        url="http://localhost:7071/mgmt/groups",
        method="POST",
        body=payload,
        status_code=201
    )
    results = p.results


replicas_sites = ",".join([r.alias for r in roles["cheops"][:nb_replicas]])
print(replicas_sites)

time.sleep(15)

with en.actions(roles=roles["cheops"][0], gather_facts=False) as p:
    p.lineinfile(
        task_name="Edit simple-pod.yml to add locations",
        search_string="    locations:",
        line=f"    locations: {replicas_sites}",
        path=f"{cheops_location}/simple-pod.yml"
    )
    p.shell(
        task_name="Apply pod config",
        cmd=f"kubectl --kubeconfig ~/.kube/config.proxified apply -f {cheops_location}/simple-pod.yml --server-side=true > /tmp/results/apply_pod_$(date +'%Y-%m-%d_%H-%M-%S').log",
        ignore_errors=True
    )
    results = p.results

    r = results.filter(task="shell")

time.sleep(20)


with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.shell(
        task_name="Get pods",
        cmd="kubectl get pods > /tmp/results/get_pods_$(date +'%Y-%m-%d_%H-%M-%S').log"
    )
    p.shell(
        task_name="Describe pod",
        cmd="kubectl describe pod simpleapp-pod > /tmp/results/describe_pods_$(date +'%Y-%m-%d_%H-%M-%S').log",
        ignore_errors=True
    )



# Pinging to check network constraints
with en.actions(roles=roles["cheops"], gather_facts=False) as p:
    for node in roles["cheops"]:
        p.shell(
            task_name="Pings before cut",
            cmd=f"ping -c 5 {node.address} >> /tmp/results/ping_before_cut_$(date +'%Y-%m-%d_%H-%M-%S').log"
        )


with en.actions(roles=roles["cheops"][1], gather_facts=False) as p:
    p.lineinfile(
        task_name="Edit simple-pod.yml to update",
        search_string="    app.kubernetes.io/name: SimpleApp",
        line="    app.kubernetes.io/name: UpdatedSimpleApp",
        path=f"{cheops_location}/simple-pod.yml"
    )
    p.shell(
        task_name="Apply pod config",
        cmd=f"kubectl --kubeconfig ~/.kube/config.proxified apply -f {cheops_location}/simple-pod.yml --server-side=true > /tmp/results/update_pod_$(date +'%Y-%m-%d_%H-%M-%S').log",
        ignore_errors=True
    )
    results = p.results

    r = results.filter(task="shell")

time.sleep(40)


with en.actions(roles=roles["cheops"], gather_facts=True) as p:
    p.shell(
        task_name="Get pods",
        cmd="kubectl get pods > /tmp/results/get_pods_after_update_$(date +'%Y-%m-%d_%H-%M-%S').log"
    )
    p.shell(
        task_name="Describe pod",
        cmd="kubectl describe pod simpleapp-pod > /tmp/results/describe_pods_after_update_$(date +'%Y-%m-%d_%H-%M-%S').log",
        ignore_errors=True
    )



# Get results files from nodes after the experiments
## Tar results on nodes
with en.actions(roles=roles["cheops"], gather_facts=False) as p:
    p.archive(
        task_name="Archive results",
        path="/tmp/results/",
        dest="/tmp/results.tar.gz"
    )


## Fetch results in appropriate files
## In a results folder, we create a folder with the name of the experiment and the date
## Each node results will be in a separate tar.gz in this folder
filename = os.path.splitext(os.path.basename(__file__))[0]
backup_dir = "results/"+ filename + "_" + datetime.datetime.now().strftime("%Y-%m-%d_%H-%M")
back_dir = os.path.abspath(backup_dir)
os.path.isdir(back_dir) or os.mkdir(back_dir)

with en.actions(roles=roles["cheops"], gather_facts=False) as p:
    p.fetch(
        task_name="Fetching results",
        src="/tmp/results.tar.gz",
        dest=back_dir+"/{{ inventory_hostname_short }}.tar.gz",
        flat="yes"
    )


## Extract files into appropriate folders (same name without the .tar.gz)
for f in os.listdir(back_dir):
    if f.endswith("tar.gz"):
        f_path= back_dir + "/" + f
        where_dot = f_path.index('.')
        folder_name = f_path[:where_dot]
        print(folder_name)
        tar = tarfile.open(f_path, "r:gz")
        tar.extractall(folder_name)
        tar.close()


with open(back_dir+'/experiment-info.txt', 'a') as f:
    f.write("Replicas sites: " + replicas_sites + "\n")
    f.write("Faulty sites: " + roles["faulty"][0].alias + "\n")
