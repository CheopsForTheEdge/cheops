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
import firewall_block
import g5k

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
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 13"),
                'sites': (None, g5k.sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli incrby {id} -12"),
                'sites': (None, g5k.sites),
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
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            firewall_block.activate(g5k.roles_for_hosts)

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 13"),
                'sites': (None, g5k.sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli incrby {id} -12"),
                'sites': (None, g5k.sites),
                'type': (None, 'inc'),
            })

            firewall_block.deactivate(g5k.roles_for_hosts)
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
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
                'config': (None, json.dumps(config)),
            })
            self.wait_and_verify(id)

            firewall_block.activate(g5k.roles_for_hosts)

            self.do(id, 0, {
                'command': (None, f"redis-cli set {id} 29"),
                'sites': (None, g5k.sites),
                'type': (None, 'set'),
            })
            self.do(id, 1, {
                'command': (None, f"redis-cli incrby {id} 12"),
                'sites': (None, g5k.sites),
                'type': (None, 'inc'),
            })
            self.do(id, 2, {
                'command': (None, f"redis-cli sadd {id} error"),
                'sites': (None, g5k.sites),
                'type': (None, 'inc'),
            })

            firewall_block.deactivate(g5k.roles_for_hosts)
            self.wait_and_verify(id)

            self.verify_shell(f"redis-cli -c get {id}")

if __name__ == '__main__':
    g5k.init()
    firewall_block.deactivate(g5k.roles_for_hosts)

    import unittest
    unittest.main()
