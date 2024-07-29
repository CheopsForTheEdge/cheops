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
from prelude import *
import firewall_block

class TestRedis(tests.CheopsTest):
    def test_simple(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            config = {'RelationshipMatrix': [
                {'Before': 'set', 'After': 'set', 'Result': 'take-one'},
                {'Before': 'set', 'After': 'inc', 'Result': 'take-both-keep-order'},
                {'Before': 'inc', 'After': 'set', 'Result': 'take-both-reverse-order'},
            ]}

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 23"),
                'sites': (None, sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 12"),
                'sites': (None, sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli incrby {id}"),
                'sites': (None, sites),
                'type': (None, 'inc'),
            })

            self.wait_and_verify(id)

            self.verify_shell(f"redis-cli -c get {id}")

    def test_simple_with_disconnect(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            config = {'RelationshipMatrix': [
                {'Before': 'set', 'After': 'set', 'Result': 'take-one'},
                {'Before': 'set', 'After': 'inc', 'Result': 'take-both-keep-order'},
                {'Before': 'inc', 'After': 'set', 'Result': 'take-both-reverse-order'},
            ]}

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 23"),
                'sites': (None, sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            firewall_block.activate([roles_for_hosts[2]])

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 12"),
                'sites': (None, sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli incrby {id}"),
                'sites': (None, sites),
                'type': (None, 'inc'),
            })

            firewall_block.deactivate([roles_for_hosts[2]])
            self.wait_and_verify(id)

            self.verify_shell(f"redis-cli -c get {id}")

    def test_simple_with_failure(self):
        id = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(id=id):
            config = {'RelationshipMatrix': [
                {'Before': 'set', 'After': 'set', 'Result': 'take-one'},
                {'Before': 'set', 'After': 'inc', 'Result': 'take-both-keep-order'},
                {'Before': 'inc', 'After': 'set', 'Result': 'take-both-reverse-order'},
            ]}

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 23"),
                'sites': (None, sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            firewall_block.activate([roles_for_hosts[2]])

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 12"),
                'sites': (None, sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli sadd {id} error"),
                'sites': (None, sites),
                'type': (None, 'set'),
            })

            firewall_block.deactivate([roles_for_hosts[2]])
            self.wait_and_verify(id)

            self.verify_shell(f"redis-cli -c get {id}")

if __name__ == '__main__':
    import unittest
    unittest.main()
