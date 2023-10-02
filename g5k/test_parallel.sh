#!/usr/bin/env sh

. ./env.sh

n1=$(id -un)_NODE_1
nn1=$(printenv $n1)
n2=$(id -un)_NODE_2
nn2=$(printenv $n2)
n3=$(id -un)_NODE_3
nn3=$(printenv $n3)

sed "s/LOCATIONS/$nn1,$nn2,$nn3/ ; s/REPLICAS/1/ ; s/PORT/80/" ../simple-deployment.yml.tpl > simple-deployment.yml
curl -s "http://$nn1:8079" --data-binary @simple-deployment.yml

read -p "Continue ? "

sed "s/LOCATIONS/$nn1,$nn2,$nn3/ ; s/REPLICAS/2/ ; s/PORT/80/" ../simple-deployment.yml.tpl > simple-deployment-replicas.yml
sed "s/LOCATIONS/$nn1,$nn2,$nn3/ ; s/REPLICAS/1/ ; s/PORT/90/" ../simple-deployment.yml.tpl > simple-deployment-port.yml

curl -s "http://$nn2:8079" --data-binary @simple-deployment-replicas.yml &
curl -s "http://$nn3:8079" --data-binary @simple-deployment-port.yml &

fg
fg
