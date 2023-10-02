#!/usr/bin/env sh

. ./env.sh

n1=$(id -un)_NODE_1
nn1=$(printenv $n1)
n2=$(id -un)_NODE_2
nn2=$(printenv $n2)
n3=$(id -un)_NODE_3
nn3=$(printenv $n3)

LOCATIONS="-H 'X-Cheops-Location: $nn1' -H 'X-Cheops-Location: $nn2' -H 'X-Cheops-Location: $nn3'"

sed "s/REPLICAS/1/ ; s/PORT/80/" ../simple-deployment.yml.tpl > simple-deployment.yml
eval "curl -s $LOCATIONS \"http://$nn1:8079\" --data-binary @simple-deployment.yml | jq '.'"

read -p "Continue ? "

sed "s/REPLICAS/2/ ; s/PORT/80/" ../simple-deployment.yml.tpl > simple-deployment-replicas.yml
sed "s/REPLICAS/1/ ; s/PORT/90/" ../simple-deployment.yml.tpl > simple-deployment-port.yml

(eval "curl -s $LOCATIONS \"http://$nn2:8079\" --data-binary @simple-deployment-replicas.yml | jq '.'") &
(eval "curl -s $LOCATIONS \"http://$nn3:8079\" --data-binary @simple-deployment-port.yml | jq '.'") &
