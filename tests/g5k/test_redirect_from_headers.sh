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

LOCATIONS="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"

sed "s/REPLICAS/1/ ; s/PORT/80/" ../simple-deployment.yml.tpl > simple-deployment.yml
eval "curl -s $LOCATIONS \"http://$nn1:8079\" --data-binary @simple-deployment.yml | jq '.'"

read -p "Continue ? "

# Test on a site that knows nothing and with no location, should fail
sed "s/REPLICAS/1/ ; s/PORT/90/" ../simple-deployment.yml.tpl > simple-deployment-port.yml

curl -s "http://$nn4:8079" --data-binary @simple-deployment-port.yml

read -p "Continue ? "

# Test again but with location, this time should work
eval "curl -L -s $LOCATIONS \"http://$nn4:8079\" --data-binary @simple-deployment.yml | jq '.'"
