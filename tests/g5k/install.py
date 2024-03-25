#!/usr/bin/env python

import sys
import os

if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)

import socket
hostname = socket.gethostname()
if hostname == "fnantes":
    site = "nantes"
    cluster = "econome"
elif hostname == "fgrenoble":
    site = "grenoble"
    cluster = "dahu"
elif hostname == "flille":
    site = "lille"
    cluster = "chiclet"
else:
    print("unknown frontend")
    os.exit(1)

import enoslib as en

en.init_logging()

network = en.G5kNetworkConf(type="prod", roles=["my_network"], site=site)

conf = (
    en.G5kConf.from_settings(job_type=[], walltime="01:50:00", job_name="cheops")
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

    #Couch
    p.apt_key(url="https://binaries2.erlang-solutions.com/GPG-KEY-pmanager.asc")
    p.apt_repository(repo="deb https://binaries2.erlang-solutions.com/debian {{ansible_distribution_release}}-esl-erlang-25 contrib")
    p.apt(update_cache=True, pkg="esl-erlang", state="absent")
    p.apt(update_cache=True, pkg="esl-erlang")

    p.apt(pkg=["build-essential", "pkg-config", "libicu-dev", "libmozjs-78-dev"])

    p.file(
        path="/tmp/couchdb",
        state="directory"
    )
    p.uri(
        url="https://dlcdn.apache.org/couchdb/source/3.3.3/apache-couchdb-3.3.3.tar.gz",
        dest="/tmp/couchdb",
    )

    p.file(
        path="/tmp/couchdb/apache-couchdb-3.3.3",
        state="directory"
    )
    p.unarchive(
        remote_src=True,
        src="/tmp/couchdb/apache-couchdb-3.3.3.tar.gz",
        dest="/tmp/couchdb"
    )
    p.shell(
        cmd="""
        ./configure --disable-docs --spidermonkey-version 78
        make release
        """,
        chdir="/tmp/couchdb/apache-couchdb-3.3.3"
    )

    if False:
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

    p.group(
        name="couchdb"
    )
    p.user(
        name="couchdb",
        shell="/bin/true",
        home="/opt/couchdb",
        group="couchdb",
    )
    p.copy(
        remote_src=True,
        src="/tmp/couchdb/apache-couchdb-3.3.3/rel/couchdb/",
        dest="/opt/couchdb"
    )
    p.file(
        path="/opt/couchdb",
        owner="couchdb",
        recurse=True
    )

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
    p.lineinfile(
        path="/opt/couchdb/etc/local.ini",
        line="admin = password",
        insertafter="\[admins\]"
    )

    p.copy(
        dest="/lib/systemd/system/couchdb.service",
        content="""
            [Unit]
            Description=couchdb
            After=network-online.target

            [Service]
            Restart=on-failure
            ExecStart=/opt/couchdb/bin/couchdb
            User=couchdb

            [Install]
            WantedBy=multi-user.target
        """,
    )
    p.systemd(
        name="couchdb",
        state="started"
    )



    # base and go
    p.apt(
        update_cache=True,
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release", "golang-1.19"]
    )

# k3s
for role in roles["cheops"]:
    k3s = en.K3s(master=[role], agent=[])
    k3s.deploy()
