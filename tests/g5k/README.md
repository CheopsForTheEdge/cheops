# Testing on g5k

The current directory contains a few integration tests. Feel free to take inspiration from them

To instantiate machines, run ./install.py.
To clean the content and (re-)start cheops, run ./rerun.sh.
To easily see the logs, run ./multitail.sh

After that any test can be run, through make, or individually. See the header of each test to understand what it does

# Explanation of files

```
.
├── dump_garbage.sh			# displays deleted resources
├── dump_replications.sh	# displays replication jobs (couchdb)
├── dump_resources.sh		# displays known resources grouped by node if similar
├── example_use.sh			# shows how to use the cli
├── env.sh					# creates ~/.oarnodes and ~/.oarnodes.json (for chephren) with list of nodes
├── install.py				# g5k script to create nodes
├── Makefile				# to easily run everything
├── multitail.sh			# tails logs of all cheops processes on all machines
├── rerun.sh				# reload script to start afresh
├── restart.sh				# individual reload script for a node
├── g5k.py					# helper for setting g5k-related variables
├── tests.py				# base class for tests
├── firewall_block.py		# helper for tests managing the fake disconnection
└── test_*.py/sh			# tests with different scenarios and apps
```

## g5k.py

This file creates the appropriate grid5000 resources in enoslib
(https://discovery.gitlabpages.inria.fr/enoslib/index.html) so that they can be
reused in later files.

If the cluster, the number of machines, or any paramaters need to be changed it
is best to do it here.

## install.py

This script reuses resources defined in g5k.py and installs and configures the
necessary components on _all_ nodes:
- A proper version of go (1.19, the one in bullseye is way too old)
- CouchDB setup as single nodes (ie not cluster), listening to the whole world,
started and restarting thanks to a systemd service
- Kubernetes as an example application
- Redis as another example application, for some tests, installed as standalone nodes

In progress in this file:
- There is some code to install redis as a single cluster with all nodes
- YCSB: this is the performance benchmark from Yahoo. Was to be used with redis
cluster to compare with redis+cheops. Not done yet.

This script is expected to be ran from a site in g5k, typically a frontend.

## rerun.sh / restart.sh


`rerun.sh` is a script to be run interactively at the beginning of each
test/experiment. It runs `restart.sh` on all reserved resources. Just like
`install.py`, it is expected to be run from a frontend. In order for these
scripts to run, the whole cheops repository is expected to exist on a shared
directory: typically, just put the whole repo somewhere in $HOME in a grid5k
cluster and it will be shared with all nodes.

`restart.sh` is not expected to be manually run. It does all the cleaning on a node:
- rebuild and restart cheops
- remove all resources from kubernetes
- clean CouchDB data, remove existing CouchDB replications
- deploy the newest version of chephren-ui
- deploy the simple_app test application
- rebuilds the cli

`rerun.sh` then copies the newly built cli back to the frontend

## tests

Tests are described in `test_*.py` or `test_*.sh` files. The python files
import facilities from:
- `tests.py`: a base class with helpers for running and checking things
- `firewall_block.py`: helpers for simulating "cut" connections and re-plug
them, by blocking all communications from node to node on port 5984: that's the
CouchDB port, meaning that no synchronization happens anymore. It is still
possible to communicate locally, so Cheops still knows how to talk to its local
CouchDB node. This file also has a helper that waits that all operations have
been synchronized, that they have all been analyzed by Cheops, and that the
resulting set of operations have been run. Basically, this is the state where
nothing more can happen until a new operation comes in.

Note that tests are not proper integration tests. They not only tests through
Cheops return content, but also by directly querying CouchDB.

At the time of writing 3 applications are tested:
- filesystem in `test_parallel.py`
- kubernetes in `test_kube.py`
- redis in `test_redis.py`

Those 3 tests are the basis of our continuous testing strategy. They define the
kind of behaviour we are interested in. Any other application should take
inspiration from them to implement the same tests.

## multitail.sh

A helper script that runs multitail, a tool to follow multiple files as the
name implies. The script automatically sets it up to follow all cheops systemd
logs on all nodes. It is expected to be run from a frontend.

Further details on multitail can be found on its site:
https://vanheusden.com/multitail/