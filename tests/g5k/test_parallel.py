

import sys
import os
import enoslib as en

def pp(arg):
    print(json.dumps(arg, indent=4))

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
else:
    site = "nantes"
    cluster = "econome"

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



import random, string

locations_header = {'X-Cheops-Location': ', '.join([r.alias for r in roles[:3]])}
id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))


import requests
r1 = requests.post(f"http://{hosts[0]}:8079/{id}", data='mkdir -p /tmp/foo; ls /tmp/foo', headers=locations_header)
assert r1.status_code == 200


import json

replies = [requests.get(f"http://{hosts[0]}:5984/cheops/{id}") for host in hosts]
for reply in replies:
    assert reply.status_code == 200
contents = [reply.json() for reply in replies]
for content in contents:
    assert len(content['Units']) == 1

with en.actions(roles=roles[:3]) as p:
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

r = requests.post(f"http://{hosts[0]}:8079/{id}", data='echo left > /tmp/foo/content', headers=locations_header)
assert r.status_code == 200

r = requests.post(f"http://{hosts[1]}:8079/{id}", data='echo right > /tmp/foo/content', headers=locations_header)
assert r.status_code == 200

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

replies = [requests.get(f"http://{hosts[0]}:5984/cheops/{id}") for host in hosts]
for reply in replies:
    assert reply.status_code == 200
contents = [reply.json() for reply in replies]
units = [content['Units'] for content in contents]
for u in units[1:]:
    assert len(u) == 3
    # Make sure we have the same content everywhere
    assert u == units[0]

    assert u[0]['Generation'] == 1
    assert u[1]['Generation'] == 2
    assert u[2]['Generation'] == 2


for host in hosts[:3]:
    query = {"selector": {
        "Type": "REPLY",
        "Site": host,
        "ResourceId": id
    }}
    r = requests.post(f"http://{host}:5984/cheops/_find", data=json.dumps(query), headers={"Content-Type": "application/json"})
    for doc in r.json()['docs']:
        assert doc['Status'] == "OK"
