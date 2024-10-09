#!/usr/bin/env python

import g5k
import enoslib as en

g5k.init()

print(g5k.roles)

with en.actions(roles=g5k.roles, gather_facts=True) as p:
    p.apt(update_cache=True)

    #Couch
    p.apt_key(url="https://binaries2.erlang-solutions.com/GPG-KEY-pmanager.asc")
    p.apt_repository(repo="deb https://binaries2.erlang-solutions.com/debian {{ansible_distribution_release}}-esl-erlang-25 contrib")
    p.apt(pkg="esl-erlang", state="absent")
    p.apt(pkg="esl-erlang")

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



    # base, go and redis
    p.apt(
        pkg=["apt-transport-https", "ca-certificates", "curl", "gnupg", "lsb-release", "golang-1.19", "redis"]
    )

    # redis
    p.lineinfile(
            path="/etc/redis/redis.conf",
            search_string="bind 127.0.0.1",
            line="bind 0.0.0.0 ::"
    )

    if False:
        p.lineinfile(
                path="/etc/redis/redis.conf",
                search_string="# cluster-config-file",
                line="cluster-config-file nodes.conf"
        )
        p.lineinfile(
                path="/etc/redis/redis.conf",
                search_string="# cluster-enabled",
                line="cluster-enabled yes"
        )
        p.lineinfile(
                path="/etc/redis/redis.conf",
                search_string="appendonly no",
                line="appendonly yes"
        )
        p.lineinfile(
                path="/etc/redis/redis.conf",
                search_string="# cluster-node-timeout",
                line="cluster-node-timeout 5000"
        )
    p.systemd(
            name="redis",
            state="restarted"
    )

# Redis cluster
if False:
    results = en.run_command("redis-cli cluster info", roles=g5k.roles[:1])
    stdout = results[0].stdout
    if 'cluster_state:ok' not in stdout:
        all_facts = en.gather_facts(roles=g5k.roles)
        facts = all_facts['ok']
        addresses = {}
        for host in facts:
            addresses[host] = facts[host]['ansible_default_ipv4']['address']

        redis_hosts = [f"{addresses[role.alias]}:6379" for role in g5k.roles]
        en.run_command(f"redis-cli --cluster create {' '.join(redis_hosts)} --cluster-replicas 1 --cluster-yes", roles = roles[:1])

# YCSB
with en.actions(roles=g5k.roles, gather_facts=True) as p:
    p.file(
        path="/opt/ycsb",
        state="directory"
    )
    p.uri(
            url="https://github.com/brianfrankcooper/YCSB/releases/download/0.17.0/ycsb-0.17.0.tar.gz",
        dest="/tmp"
    )
    p.file(
        path="/tmp/ycsb-0.17.0",
        state="directory"
    )
    p.unarchive(
        remote_src=True,
        src="/tmp/ycsb-0.17.0.tar.gz",
        dest="/opt/ycsb"
    )
    

# k3s
for role in g5k.roles:
    k3s = en.K3s(master=[role], agent=[])
    k3s.deploy()
