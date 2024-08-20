#!/usr/bin/env python

# This test
# - creates a resource
# - lets it be synchronized
# - blocks synchronization
# - updates the resource on 2 different nodes
# - restores synchronization and lets it run
#
# After that we should have the same resource everywhere

import random
import string
import json
import enoslib as en

import tests
import requests
from prelude import *
import firewall_block

class TestParallel(tests.CheopsTest):
    def init(self, id, request):
        self.do(id, 0, request)

        replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts[:3]]
        for reply in replies:
            self.assertEqual(200, reply.status_code)
            self.assertEqual(replies[0].json(), reply.json())

        # Deactivate sync (by blocking at firewall level), send 2 parallel, conflicting commands, and reactivate sync
        firewall_block.activate(roles_for_hosts)

    def do_left_and_right(self, id, request_left, request_right):
        self.do(id, 0, request_left)
        self.do(id, 1, request_right)

        firewall_block.deactivate(roles_for_hosts)
        self.wait_and_verify(id)

    def test_simple(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/init"),
                'sites': (None, sites),
                'type': (None, "touch"),
            })
            self.do_left_and_right(id, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/left"),
                'sites': (None, sites),
                'type': (None, "touch"),
            }, {
                'command': (None, f"mkdir -p /tmp/{id} && touch /tmp/{id}/right"),
                'sites': (None, sites),
                'type': (None, "touch"),
            })
            self.verify_shell(f"ls /tmp/{id}")

    def test_set_and_add(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo init > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': "take-one"},
                        {'Before': 'set', 'After': 'add', 'Result': "take-both-keep-order"},
                        {'Before': 'add', 'After': 'set', 'Result': "take-both-reverse-order"},
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
            self.verify_shell(f"cat /tmp/{id}/file")

    def test_set_and_set(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.init(id, {
                'command': (None, f"mkdir -p /tmp/{id} && echo init > /tmp/{id}/file"),
                'sites': (None, sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': "take-one"},
                        {'Before': 'set', 'After': 'add', 'Result': "take-both-keep-order"},
                        {'Before': 'add', 'After': 'set', 'Result': "take-both-reverse-order"},
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
            self.verify_shell(f"cat /tmp/{id}/file")

if __name__ == '__main__':
    import unittest
    unittest.main()
