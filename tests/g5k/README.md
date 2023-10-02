# Testing on g5k

The current directory contains the files to test a few scenarios on Grid5000.

First step: install all the necessary tools using the jupyter notebook 'cheops with kube.ipynb'.

Run multitail.sh in another tmux pane to follow activity of the nodes (they will be autoconfigured)

After that there are multiple available tests:

# test_parallel.sh

Needs 3 nodes. 

- Runs a first deployment
- asks before continuing
- then 2 changes for the same resource on 2 different nodes in parallel.

Expected result: the 2 new changes are merged and deployed on all sites


# test_different_sites.sh

Needs 4 nodes.

- Runs a first deployment
- asks before continuing
- then a different deployment on another set of nodes.

Expected result: the 2 deployments exist in parallel where they are supposed to


# test_redirect.sh

Needs 4 nodes.

- Runs a first deployment
- asks before continuing
- then tries to run a change on the same resource but from fourth node: fourth node has never seen anyone so doesn't know who to redirect to
- asks before continuing
- tries again, this time with locations in the header.

Expected result: the client is redirected and the change is applied
