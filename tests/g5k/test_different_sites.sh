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

LOCATIONS_1="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"
LOCATIONS_2="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn4'"

eval "curl -s $LOCATIONS_1 \"http://$nn1:8079/left\" --data-binary 'mkdir -p /tmp/foo; touch /tmp/foo/left' | jq '.'"

read -p "Continue ? "

eval "curl -s $LOCATIONS_2 \"http://$nn1:8079/right\" --data-binary 'mkdir -p /tmp/foo; touch /tmp/foo/right' | jq '.'"
