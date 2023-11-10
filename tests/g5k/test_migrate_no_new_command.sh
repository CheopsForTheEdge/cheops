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

id=$(cat /dev/urandom | head -c 20 | base32)

curl -v -H "X-Cheops-Location: $nn1" -H "X-Cheops-Location: $nn2" -H "X-Cheops-Location: $nn3" "http://$nn1:8079/$id" --data-binary 'mkdir /tmp/foo > /dev/null'

read -p "Continue ? "

curl -v -H "X-Cheops-Location: $nn1" -H "X-Cheops-Location: $nn2" -H "X-Cheops-Location: $nn4" "http://$nn1:8079/$id" --data-binary 'mkdir /tmp/foo > /dev/null'

echo Expected: no new command with different sites
