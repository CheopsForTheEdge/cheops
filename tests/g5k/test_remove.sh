#!/usr/bin/env sh

. ./env.sh

n1=$(id -un)_NODE_1
nn1=$(printenv $n1)
n2=$(id -un)_NODE_2
nn2=$(printenv $n2)
n3=$(id -un)_NODE_3
nn3=$(printenv $n3)

LOCATIONS="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"

sed "s/REPLICAS/1/ ; s/PORT/80/" simple-deployment.yml.tpl > simple-deployment.yml
eval "curl -s $LOCATIONS \"http://$nn1:8079\" --data-binary @simple-deployment.yml | jq '.'"

read -p "Continue ? "

idtodelete=$(curl -s -H "Content-Type: application/json" "http://$nn1:5984/cheops/_find" --data-binary '{"selector": {"Payload.Method": "POST"}}' | jq -r '.docs[0] | ._id')
doc=$(curl -s http://$nn1:5984/cheosp/$idtodelete)
deleteddoc=$(echo $doc | jq '.Deleted = true')
revtodelete=$(echo $doc | jq -r '._rev')

echo removing $idtodelete $revtodelete

echo $deleteddoc | curl -s "http://$nn1:5984/cheops/${idtodelete}?rev=${revtodelete}" --data-binary -
