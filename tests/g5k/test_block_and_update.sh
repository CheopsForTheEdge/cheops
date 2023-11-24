#!/usr/bin/env sh

set -euxo pipefail

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
id=$(head -c 20 /dev/urandom | base32)
echo "id is $id"

eval "curl -s $LOCATIONS \"http://$nn1:8079/$id\" --data-binary 'mkdir -p /tmp/foo; touch /tmp/foo/left' | jq '.' > 1"

echo "sleeping a few secs"
sleep 5

ssh $nn2 sudo nft add rule ip filter INPUT ip saddr 127.0.0.1 accept
ssh $nn2 sudo nft add rule ip filter INPUT tcp dport 5984 drop

curl -s "http://$nn1:8079/$id" --data-binary 'mkdir -p /tmp/foo; touch /tmp/foo/right' | jq '.' > 2

echo "status during the block"
for node in $nn1 $nn2 $nn3
do
				echo $node
				ssh $node curl -s "http://localhost:5984/cheops/$id" 
done

h1=$(ssh $nn2 sudo nft --handle list chain ip filter INPUT | awk '/ip saddr 127.0.0.1 accept/ {print $NF}')
h2=$(ssh $nn2 nft --handle list chain ip filter INPUT | awk '/tcp dport 5984 drop/ {print $NF}')

ssh $nn2 sudo nft delete rule ip filter INPUT handle $h1
ssh $nn2 sudo nft delete rule ip filter INPUT handle $h2

while true
do
				code=$(curl -m 1 -s -w http_code "http://$nn2:5984")
				if [ $code -eq 200 ]; then break; fi
done

echo "status after the block"
for node in $nn1 $nn2 $nn3
do
				echo $node
				ssh $node curl -s "http://localhost:5984/$id" 
done

