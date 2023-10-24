# Testing on g5k

The current directory contains the files to test a few scenarios on Grid5000.

From the this directory, run the following:

```
# Book machines
$ ./install.py

# Source booked machines
$ . env.sh

# Redeploy cheops, clean the database and the current state
$ ./rerun.sh

# In one terminal, look at the output of the logs
$ ./multitail.sh

# In another, run the experiment you want
$ ./test_parallel.sh
$ ./test_migrate_no_new_command.sh
```

There are multiple available tests:

## test_parallel.sh

Needs 3 nodes.

- Runs a first deployment
- asks before continuing
- then 2 changes for the same resource on 2 different nodes in parallel.

Expected result: the 2 new changes are merged and deployed on all sites

## test_different_sites.sh

Needs 4 nodes.

- Runs a first deployment
- asks before continuing
- then a different deployment on another set of nodes.

Expected result: the 2 deployments exist in parallel where they are supposed to
