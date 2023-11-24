#!/usr/bin/env sh

. ./env.sh

n1=$(id -un)_NODE_1
nn1=$(printenv $n1)
n2=$(id -un)_NODE_2
nn2=$(printenv $n2)
n3=$(id -un)_NODE_3
nn3=$(printenv $n3)

LOCATIONS="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"

eval "curl -s $LOCATIONS \"http://$nn1:8079/id\" --data-binary 'mkdir -p /tmp/foo' | jq '.'"

read -p "Continue ? "

(eval "curl -s $LOCATIONS \"http://$nn2:8079/id\" --data-binary 'echo left > /tmp/foo/left' | jq ''") &
(eval "curl -s $LOCATIONS \"http://$nn3:8079/id\" --data-binary 'echo right > /tmp/foo/right' | jq '.'") &
