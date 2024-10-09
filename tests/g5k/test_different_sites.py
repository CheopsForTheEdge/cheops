#!/usr/bin/env python

# This test creates 2 different resources on 2 different but overlapping sets of locations.
# We make sure everything is where it is supposed to be and not anywhere else

import random
import string
import json
import enoslib as en

import tests
import firewall_block
import g5k

class TestDifferentSites(tests.CheopsTest):
    def test_simple(self):
        id1 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        id2 = ''.join(random.choice(string.ascii_lowercase) for i in range(10))
        with self.subTest(ids=[id1,id2], hosts=g5k.hosts):
            self.do(id1, 0, {
                'command': (None, "mkdir -p /tmp/foo; echo left > /tmp/foo/left"),
                'sites': (None, '&'.join(g5k.hosts[:3])),
                'type': (None, 'mkdir'),
            })

            self.do(id2, 1, {
                'command': (None, "mkdir -p /tmp/foo; echo right > /tmp/foo/right"),
                'sites': (None, '&'.join(g5k.hosts[1:])),
                'type': (None, 'mkdir'),
            })
            self.wait_and_verify(id1)
            self.wait_and_verify(id2, g5k.hosts[1:])

if __name__ == '__main__':
    g5k.init()
    firewall_block.deactivate(g5k.roles_for_hosts)

    import unittest
    unittest.main()
