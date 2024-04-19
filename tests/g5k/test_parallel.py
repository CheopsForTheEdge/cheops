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
import unittest
import random, string
import json
import requests
import socket
import enoslib as en

import firewall_block

# Hack
if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)


# Make it work on nantes and grenoble
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
sites = '&'.join(hosts[:3])
roles_for_hosts = [role for role in roles if role.alias in hosts[:3]]

# Ensure firewall allows sync
firewall_block.deactivate(roles_for_hosts)

class TestParallel(unittest.TestCase):
    def init(self, id, request):
        r1 = requests.post(f"http://{hosts[0]}:8079/exec/{id}", files=request)
        self.assertEqual(200, r1.status_code, id)

        replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
        for reply in replies:
            self.assertEqual(200, reply.status_code)
            self.assertEqual(replies[0].json(), reply.json())

        # Deactivate sync (by blocking at firewall level), send 2 parallel, conflicting commands, and reactivate sync
        firewall_block.activate(roles_for_hosts)

    def do_left_and_right(self, id, request_left, request_right):
        r = requests.post(f"http://{hosts[0]}:8079/exec/{id}", files=request_left)
        self.assertEqual(200, r.status_code)
        r = requests.post(f"http://{hosts[1]}:8079/exec/{id}", files=request_right)
        self.assertEqual(200, r.status_code)

        firewall_block.deactivate(roles_for_hosts)

        # Once content is synchronized, make sure it is actually the same
        replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
        for reply in replies:
            self.assertEqual(200, reply.status_code)
            self.assertEqual(replies[0].json(), reply.json())
        contents = [reply.json() for reply in replies]
        for content in contents:
            self.assertEqual(3, len(content['Operations']))
            self.assertEqual(content['Operations'], contents[0]['Operations'])

        # Make sure the replies are all ok
        for host in hosts[:3]:
            query = {"selector": {
                "Type": "REPLY",
                "Site": host,
                "ResourceId": id
            }}
            r = requests.post(f"http://{host}:5984/cheops/_find", json=query, headers={"Content-Type": "application/json"})
            for doc in r.json()['docs']:
                self.assertEqual("OK", doc['Status'])

    def verify(self, command):
        # Make sure the directory has the correct content everywhere
        with en.actions(roles=roles_for_hosts) as p:
            p.shell(command)
            results = p.results

        contents = [content.payload['stdout'] for content in results.filter(task="shell")]
        for content in contents[1:]:
            self.assertEqual(contents[0], content)

    def test_simple(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/init"),
                'sites': (None, sites),
                'type': (None, 1),
            })
            self.do_left_and_right(id, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/left"),
                'sites': (None, sites),
                'type': (None, "1"),
            }, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/right"),
                'sites': (None, sites),
                'type': (None, "1"),
            })
            self.verify(f"ls /tmp/{id}")

    def test_set_and_add(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo init > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': [1]},
                        {'Before': 'set', 'After': 'add', 'Result': [1, 2]},
                        {'Before': 'add', 'After': 'set', 'Result': [2]},
                    ]
                })),
            })
            self.do_left_and_right(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo left >> /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "add"),
            }, {
                'command': (None, f"mkdir -p /tmp/{id} && echo right >> /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "add"),
            })
            self.verify(f"cat /tmp/{id}/file")

    def test_set_and_set(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo init > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': [1]},
                        {'Before': 'set', 'After': 'add', 'Result': [1, 2]},
                        {'Before': 'add', 'After': 'set', 'Result': [2]},
                    ]
                })),
            })
            self.do_left_and_right(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo left > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
            }, {
                'command': (None, f"mkdir -p /tmp/{id} && echo right > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
            })
            self.verify(f"cat /tmp/{id}/file")

if __name__ == '__main__':
    unittest.main()
