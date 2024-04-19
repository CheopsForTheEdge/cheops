#!/usr/bin/env python

# This test creates 2 different resources on 2 different but overlapping sets of locations.
# We make sure everything is where it is supposed to be and not anywhere else

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

# Ensure firewall allows sync
with en.actions(roles=roles[:3]) as p:
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
id1 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
id2 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))

# Apply 2 different commands on 2 different ids
import requests
r1 = requests.post(f"http://{hosts[0]}:8079/exec/{id1}", files={
    'command': (None, 'mkdir -p /tmp/foo; echo left > /tmp/foo/left'),
    'sites': (None, '&'.join([h for h in hosts[:3]])),
    'type': (None, "1"),
})
assert r1.status_code == 200
r2 = requests.post(f"http://{hosts[1]}:8079/exec/{id2}", files={
    'command': (None, 'mkdir -p /tmp/foo; echo right > /tmp/foo/right'),
    'sites': (None, '&'.join([h for h in hosts[1:]])),
    'type': (None, "1"),
})
assert r2.status_code == 200

synchronization.wait(hosts)

# Check everything is there
# For first resource
for host in hosts[:3]:
    query = {
        "selector": {
            "Type": "RESOURCE",
            "ResourceId": id1
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, json=query)
    assert r.status_code == 200
    docs = r.json()['docs']
    assert len(docs) == 1
    doc = docs[0]
    assert len(doc['Operations']) == 1
    assert doc['Locations'] == hosts[:3]
    assert 'left' in doc['Operations'][0]['Command']['Command']
    assert 'right' not in doc['Operations'][0]['Command']['Command']

    query = {
        "selector": {
            "Type": "REPLY",
            "ResourceId": id1
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, json=query)
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 3
    for doc in docs:
        assert doc['Locations'] == hosts[:3]

# For second resource
for host in hosts[1:]:
    query = {
        "selector": {
            "Type": "RESOURCE",
            "ResourceId": id2
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, json=query)
    assert r.status_code == 200
    docs = r.json()['docs']
    assert len(docs) == 1
    doc = docs[0]
    assert len(doc['Operations']) == 1
    assert doc['Locations'] == hosts[1:]
    assert 'right' in doc['Operations'][0]['Command']['Command']
    assert 'left' not in doc['Operations'][0]['Command']['Command']

    query = {
        "selector": {
            "Type": "REPLY",
            "ResourceId": id2
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, json=query)
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 3
    for doc in docs:
        assert doc['Locations'] == hosts[1:]
