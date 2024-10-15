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
import firewall_block
import g5k

class TestParallel(tests.CheopsTest):

    def test_simple(self):
        firewall_block.deactivate(g5k.roles_for_hosts)

        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.do(id, 0, {
                'command': (None, f"echo init > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': "take-one"},
                        {'Before': 'set', 'After': 'add', 'Result': "take-both-keep-order"},
                        {'Before': 'add', 'After': 'set', 'Result': "take-both-reverse-order"},
                    ]
                })),
            })
            self.do(id, 0, {
                'command': (None, f"echo left >> /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })
            self.do(id, 1, {
                'command': (None, f"echo middle > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
            })
            self.do(id, 2, {
                'command': (None, f"echo right >> /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })

            self.wait_and_verify(id)
            self.verify_shell(f"cat /tmp/{id}")

    def test_simple_with_disconnect(self):
        firewall_block.deactivate(g5k.roles_for_hosts)

        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.do(id, 0, {
                'command': (None, f"echo init > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': "take-one"},
                        {'Before': 'set', 'After': 'add', 'Result': "take-both-keep-order"},
                        {'Before': 'add', 'After': 'set', 'Result': "take-both-reverse-order"},
                    ]
                })),
            })
            self.wait_and_verify(id)

            firewall_block.activate(g5k.roles_for_hosts)
            self.do(id, 0, {
                'command': (None, f"echo left >> /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })
            self.do(id, 1, {
                'command': (None, f"echo middle > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
            })
            self.do(id, 2, {
                'command': (None, f"echo right >> /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })

            firewall_block.deactivate(g5k.roles_for_hosts)

            self.wait_and_verify(id)
            self.verify_shell(f"cat /tmp/{id}")

    def test_simple_with_failure(self):
        firewall_block.deactivate(g5k.roles_for_hosts)

        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            self.do(id, 0, {
                'command': (None, f"echo init > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
                'config': (None, json.dumps({
                    'RelationshipMatrix': [
                        {'Before': 'set', 'After': 'set', 'Result': "take-one"},
                        {'Before': 'set', 'After': 'add', 'Result': "take-both-keep-order"},
                        {'Before': 'add', 'After': 'set', 'Result': "take-both-reverse-order"},
                    ]
                })),
            })
            self.wait_and_verify(id)

            firewall_block.activate(g5k.roles_for_hosts)
            self.do(id, 0, {
                'command': (None, f"echo left >> /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })
            self.do(id, 1, {
                'command': (None, f"echo middle > /tmp/{id}"),
                'sites': (None, g5k.sites),
                'type': (None, "set"),
            })
            self.do(id, 2, {
                'command': (None, f"echo right >> /tmp/{id}/sub-file"),
                'sites': (None, g5k.sites),
                'type': (None, "add"),
            })

            firewall_block.deactivate(g5k.roles_for_hosts)

            self.wait_and_verify(id)
            self.verify_shell(f"cat /tmp/{id}")

if __name__ == '__main__':
    g5k.init()
    firewall_block.deactivate(g5k.roles_for_hosts)

    import unittest
    unittest.main()
