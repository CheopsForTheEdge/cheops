#!/usr/bin/env python

# This test creates 2 different resources on 2 different but overlapping sets of locations.
# We make sure everything is where it is supposed to be and not anywhere else

import sys
import os
import enoslib as en

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
locations_header_1 = {'X-Cheops-Location': ', '.join([h for h in hosts[:3]])}
locations_header_2 = {'X-Cheops-Location': ', '.join([h for h in hosts[1:]])}
id1 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
id2 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))

# Apply 2 different commands on 2 different ids
import requests
r1 = requests.post(f"http://{hosts[0]}:8079/{id1}", data='mkdir -p /tmp/foo; echo left > /tmp/foo/left', headers=locations_header_1)
assert r1.status_code == 200
r2 = requests.post(f"http://{hosts[1]}:8079/{id2}", data='mkdir -p /tmp/foo; echo right > /tmp/foo/right', headers=locations_header_2)
assert r2.status_code == 200

# Wait for synchronization
import time
def is_synchronized():
    for host in hosts:
        changes = requests.get(f"http://{host}:5984/cheops/_changes")
        current = changes.json()['last_seq']

        sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"))
        for doc in sched.json()['docs']:
            synchronized = doc['info']['source_seq']
            if synchronized != current:
                return False
    return True

while True:
    if is_synchronized():
        break
    else:
        time.sleep(1)

# Check everything is there
import json

# For first resource
for host in hosts[:3]:
    r = requests.get(f"http://{host}:5984/cheops/{id1}")
    assert r.status_code == 200
    resource = r.json()
    assert resource['Locations'] == hosts[:3]
    assert len(resource['Units']) == 1
    assert 'left' in resource['Units'][0]['Body']
    assert 'right' not in resource['Units'][0]['Body']

    query = {
        "selector": {
            "Type": "REPLY",
            "ResourceId": id1
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, data=json.dumps(query))
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 3
    for doc in docs:
        assert doc['Locations'] == hosts[:3]
        assert 'left' in doc['Input']
        assert 'right' not in doc['Input']

# For second resource
for host in hosts[1:]:
    r = requests.get(f"http://{host}:5984/cheops/{id2}")
    assert r.status_code == 200
    resource = r.json()
    assert resource['Locations'] == hosts[1:]
    assert len(resource['Units']) == 1
    assert 'right' in resource['Units'][0]['Body']
    assert 'left' not in resource['Units'][0]['Body']

    query = {
        "selector": {
            "Type": "REPLY",
            "ResourceId": id2
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, data=json.dumps(query))
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 3
    for doc in docs:
        assert doc['Locations'] == hosts[1:]
        assert 'right' in doc['Input']
        assert 'left' not in doc['Input']

