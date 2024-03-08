#!/usr/bin/env python

# This test
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

import synchronization

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
id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
sites = '&'.join(hosts[:3])

# Apply a first command, as an "init"
import requests
r1 = requests.post(f"http://{hosts[0]}:8079/{id}", files={
    'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/init"),
    'sites': (None, sites),
    'type': (None, 1),
})
assert r1.status_code == 200
synchronization.wait(hosts)

replies = [requests.post(f"http://{host}:5984/cheops/_find", json={"selector": {"Type": "RESOURCE", "ResourceId": id}}) for host in hosts[:3]]
for reply in replies:
    assert reply.status_code == 200
    assert len(reply.json()['docs']) == 1, reply.json()

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
import time
time.sleep(3)

r = requests.post(f"http://{hosts[0]}:8079/{id}", files={
    'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/left"),
    'sites': (None, sites),
    'type': (None, "1"),
})
assert r.status_code == 200

r = requests.post(f"http://{hosts[1]}:8079/{id}", files={
    'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/right"),
    'sites': (None, sites),
    'type': (None, "1"),
})
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
synchronization.wait(hosts)

# Once content is synchronized, make sure it is actually the same
replies = [requests.post(f"http://{host}:5984/cheops/_find", json={"selector": {"Type": "RESOURCE", "ResourceId": id}}) for host in hosts[:3]]
for reply in replies:
    assert reply.status_code == 200
contents = [reply.json()['docs'][0] for reply in replies]
for content in contents:
    if content['Site'] == hosts[0]:
        assert len(content['Operations']) == 2
    elif content['Site'] == hosts[1]:
        assert len(content['Operations']) == 1


# Make sure the replies are all ok
import json
for host in hosts[:3]:
    query = {"selector": {
        "Type": "REPLY",
        "Site": host,
        "ResourceId": id
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", json=query, headers={"Content-Type": "application/json"})
    for doc in r.json()['docs']:
        assert doc['Status'] == "OK"

# Make sure the file has the correct content everywhere
with en.actions(roles=roles_for_hosts) as p:
    p.shell(f"ls /tmp/{id}")
    results = p.results

contents = [content.payload['stdout'] for content in results.filter(task="shell")]
for content in contents[1:]:
    assert content == contents[0], f"content={content} contents[0]={contents[0]}"
