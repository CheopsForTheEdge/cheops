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
locations_header_1 = {'X-Cheops-Location': ', '.join([h for h in hosts[:3]])}
id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))

# Create resource
import requests
r = requests.post(f"http://{hosts[0]}:8079/{id}", data='mkdir -p /tmp/foo; echo init > /tmp/foo/content', headers=locations_header_1)
assert r.status_code == 200

synchronization.wait(hosts)

# Check everything is there
import json

for host in hosts[:3]:
    r = requests.get(f"http://{host}:5984/cheops/{id}")
    assert r.status_code == 200
    resource = r.json()
    assert resource['Locations'] == hosts[:3]
    assert len(resource['Units']) == 1

    query = {
        "selector": {
            "Type": "REPLY",
            "ResourceId": id
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, data=json.dumps(query))
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 3
    for doc in docs:
        assert doc['Locations'] == hosts[:3]

# Set the Locations to the second set
locations_header_2 = {'X-Cheops-Location': ', '.join([h for h in hosts[1:]])}
r = requests.post(f"http://{hosts[1]}:8079/{id}", data='echo migrated > /tmp/foo/content', headers=locations_header_2)
assert r.status_code == 200

synchronization.wait(hosts)

# Verify it is scheduled for removing from first host
r = requests.get(f"http://{hosts[0]}:5984/cheops/{id}")
assert r.status_code == 200
query = {
    "selector": {
        "Type": "DELETE",
        "ResourceId": id
}}
r = requests.post(f"http://{hosts[0]}:5984/cheops/_find", headers={'Content-type': 'application/json'}, data=json.dumps(query))
assert r.status_code == 200
assert len(r.json()['docs']) == 1

# Verify it is now present in new places
for host in hosts[1:]:
    r = requests.get(f"http://{host}:5984/cheops/{id}")
    assert r.status_code == 200
    resource = r.json()
    assert resource['Locations'] == hosts[1:]
    assert len(resource['Units']) == 2

# Verify we have the 2 replies for the 2 units for each host in the new set
for host in hosts[1:]:
    query = {
        "selector": {
            "Type": "REPLY",
            "Site": host,
            "ResourceId": id
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", headers={'Content-type': 'application/json'}, data=json.dumps(query))
    assert r.status_code == 200

    docs = r.json()['docs']
    assert len(docs) == 2

# Verify we have the proper content everywhere we expect
roles_for_hosts = [role for role in roles if role.alias in hosts[1:]]
with en.actions(roles=roles_for_hosts) as p:
    p.shell("cat /tmp/foo/content")
    results = p.results

contents = [content.payload['stdout'] for content in results.filter(task="shell")]
for content in contents[1:]:
    assert content == contents[0]

# Verify we don't have the new content in the old locations
with en.actions(roles=[role for role in roles if role not in roles_for_hosts]) as p:
    p.shell("cat /tmp/foo/content")
    results = p.results

old_contents = [content.payload['stdout'] for content in results.filter(task="shell")]
for old_content in old_contents:
    assert old_content != contents[0]

