#!/usr/bin/env sh

. ./env.sh

n1=$(id -un)_NODE_1
nn1=$(printenv $n1)
n2=$(id -un)_NODE_2
nn2=$(printenv $n2)
n3=$(id -un)_NODE_3
nn3=$(printenv $n3)
n4=$(id -un)_NODE_4
nn4=$(printenv $n4)

LOCATIONS_BEFORE="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"
LOCATIONS_AFTER="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn4'"

id=$(cat /dev/urandom | head -c 20 | base64)

eval "curl -s $LOCATIONS_BEFORE \"http://$nn1:8079/$id\" --data-binary 'mkdir -p /tmp/foo > /dev/null' | jq '.'"

read -p "Continue ? "

eval "curl -s $LOCATIONS_AFTER \"http://$nn1:8079/$id\" --data-binary 'mkdir -p /tmp/foo > /dev/null' | jq '.'"

echo Expected: a new command with different sites
