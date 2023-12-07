#!/usr/bin/env python

# This tests
# - creates a resource
# - lets it be synchronized
# - blocks synchronization
# - updates the resource on 2 different nodes
# - restores synchronization and lets it run
#
# After that we should have the same resource everywhere

import sys
import os
import enoslib as en
import time

def wait_synchronization():
    def is_synchronized():
        for host in hosts:
            changes = requests.get(f"http://{host}:5984/cheops/_changes")
            current = changes.json()['last_seq']

            sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"))
            for doc in sched.json()['docs']:
                if 'info' not in doc or 'source_seq' not in doc['info']:
                    # Replication is just installed but not started yet, so we wait a bit more
                    return False
                synchronized = doc['info']['source_seq']
                if synchronized != current:
                    return False
        return True

    while True:
        if is_synchronized():
            break
        else:
            time.sleep(1)

# Hack
if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)

# Make it work on nantes and grenoble
import socket
hostname = socket.gethostname()
if hostname == "fnantes":
    site = "nantes"
    cluster = "econome"
elif hostname == "fgrenoble":
    site = "grenoble"
    cluster = "dahu"
else:
    site = "nantes"
    cluster = "econome"

# Get the cluster
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
rroles, networks = provider.init()
en.sync_info(rroles, networks)

roles = rroles["cheops"]
hosts = [r.alias for r in roles]
roles_for_hosts = [role for role in roles if role.alias in hosts[:3]]

# Ensure firewall allows sync
with en.actions(roles=roles_for_hosts) as p:
    p.iptables(
            chain="INPUT",
            source="127.0.0.1",
            jump="ACCEPT",
            state="absent"
    )
    p.iptables(
            chain="INPUT",
            protocol="tcp",
            destination_port="5984",
            jump="DROP",
            state="absent"
    )


# Build useful variables that will be reused
import random, string
locations_header = {'X-Cheops-Location': ', '.join(hosts[:3])}
id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))

# Apply a first command, as an "init"
import requests
r1 = requests.post(f"http://{hosts[0]}:8079/{id}", data='mkdir -p /tmp/foo; ls /tmp/foo', headers=locations_header)
assert r1.status_code == 200
wait_synchronization()

replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
for reply in replies:
    assert reply.status_code == 200
contents = [reply.json() for reply in replies]
for content in contents:
    assert len(content['Units']) == 1

# Deactivate sync (by blocking at firewall level), send 2 parallel, conflicting commands, and reactivate sync
with en.actions(roles=roles_for_hosts) as p:
    p.iptables(
            chain="INPUT",
            source="127.0.0.1",
            jump="ACCEPT",
            state="present"
    )
    p.iptables(
            chain="INPUT",
            protocol="tcp",
            destination_port="5984",
            jump="DROP",
            state="present"
    )

# Wait for blocking to be in place
time.sleep(3)

r = requests.post(f"http://{hosts[0]}:8079/{id}", data='echo left > /tmp/foo/content', headers=locations_header)
assert r.status_code == 200

r = requests.post(f"http://{hosts[1]}:8079/{id}", data='echo right > /tmp/foo/content', headers=locations_header)
assert r.status_code == 200

with en.actions(roles=roles_for_hosts) as p:
    p.iptables(
            chain="INPUT",
            source="127.0.0.1",
            jump="ACCEPT",
            state="absent"
    )
    p.iptables(
            chain="INPUT",
            protocol="tcp",
            destination_port="5984",
            jump="DROP",
            state="absent"
    )


# After sync is re-enabled, wait for changes to be synchronized
wait_synchronization()

# Once content is synchronized, make sure it is actually the same
replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
for reply in replies:
    assert reply.status_code == 200
contents = [reply.json() for reply in replies]
units = [content['Units'] for content in contents]
for u in units[1:]:
    assert len(u) == 3
    # Make sure we have the same content everywhere
    assert u == units[0]

    # Make sure it was actually created in parallel
    assert u[0]['Generation'] == 1
    assert u[1]['Generation'] == 2
    assert u[2]['Generation'] == 2

# Make sure the replies are all ok
import json
for host in hosts[:3]:
    query = {"selector": {
        "Type": "REPLY",
        "Site": host,
        "ResourceId": id
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", data=json.dumps(query), headers={"Content-Type": "application/json"})
    for doc in r.json()['docs']:
        assert doc['Status'] == "OK"

# Make sure the file has the correct content everywhere
with en.actions(roles=roles_for_hosts) as p:
    p.shell("cat /tmp/foo/content")
    results = p.results

contents = [content.payload['stdout'] for content in results.filter(task="shell")]
for content in contents[1:]:
    assert content == contents[0]
