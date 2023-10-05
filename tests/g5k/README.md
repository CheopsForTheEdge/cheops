# Testing on g5k

The current directory contains the files to test a few scenarios on Grid5000.

From the root of the git repo, on the frontend node:

```
# Book machines
tests/g5k/book.py

# Source booked machines
. tests/g5k/env.sh

# install all the necessary tools
parallel --nonall --sshloginfile ~/.oarnodes --tag --transferfile tests/g5k/get-tip.sh exec
parallel --nonall --sshloginfile ~/.oarnodes --tag sudo install-run-couchdb.sh
parallel --nonall --sshloginfile ~/.oarnodes --tag sudo install-run-kubernetes.sh

# start from a clean slate
restart-with-tip.sh
```

In a development phase, re-run restart.sh every time there is a change

Run multitail.sh in another tmux pane to follow activity of the nodes (they will be autoconfigured)

After that there are multiple available tests:

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

## test_redirect_from_headers.sh

Needs 4 nodes.

- Runs a first deployment
- asks before continuing
- then tries to run a change on the same resource but from fourth node: fourth node has never seen anyone so doesn't know who to redirect to
- asks before continuing
- tries again, this time with locations in the header.

Expected result: the client is redirected and the change is applied
