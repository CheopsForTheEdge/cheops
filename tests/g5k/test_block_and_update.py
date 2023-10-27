#!/usr/bin/env python

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

with en.actions(roles=roles[1]) as p:
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


import requests
import random, string
import json

locations_header = {'X-Cheops-Location': ', '.join([r.alias for r in roles[:3]])}
id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))

r1 = requests.post(f"http://{roles[0].alias}:8079/{id}", data='mkdir -p /tmp/foo', headers=locations_header)
if r1.status_code == 200:
    pp(json.loads(r1.text))

print("init ok")

with en.actions(roles=roles[1]) as p:
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

print("blocked")


r2 = requests.post(f"http://{roles[0].alias}:8079/{id}", data = 'echo update > /tmp/foo/file')
if r2.status_code != 200:
    print("Error posting update")
    print(r2.content)
    sys.exit()

with en.actions(roles=roles[:3]) as p:
    p.uri(
            url=f"http://localhost:5984/cheops/{id}",
            return_content=True,
            ignore_errors=True
    )
    results = p.results

res = [{r.host: json.loads(r.payload['content'])} for r in results.filter(task="uri")]
pp(res)

sys.exit()
with en.actions(roles=roles[1]) as p:
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
print("unblocked")
