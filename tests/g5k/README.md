# Testing on g5k

The current directory contains a few integration tests. Feel free to take inspiration from them

To instantiate machines, run ./install.py.
To clean the content and (re-)start cheops, run ./rerun.sh.
To easily see the logs, run ./multitail.sh

After that any test can be run, through make, or individually. See the header of each test to understand what it does

# Explanation of files

.
├── dump_garbage.sh			# displays deleted resources
├── dump_replications.sh	# displays replication jobs (couchdb)
├── dump_resources.sh		# displays known resources grouped by node if similar
├── example_use.sh		# shows how to use the cli
├── env.sh					# creates ~/.oarnodes and ~/.oarnodes.json (for chephren) with list of nodes
├── install.py				# g5k script to create nodes
├── Makefile				# to easily run everything
├── multitail.sh			# tails logs of all cheops processes on all machines
├── rerun.sh				# reload script to start afresh
├── restart.sh				# individual reload script for a node
├── synchronization.py		# helper for tests
└── test_*.py/sh			# tests with different scenarios
