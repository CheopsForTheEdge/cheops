#!/usr/bin/env python
import unittest

import g5k
import firewall_block

g5k.init()
firewall_block.deactivate(g5k.roles_for_hosts)

tests = unittest.defaultTestLoader.discover(start_dir=".")
unittest.TextTestRunner().run(tests)
