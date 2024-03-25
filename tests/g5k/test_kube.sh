#!/usr/bin/env sh

. ./env.sh

clean() {
				ssh $host3 sudo nft delete chain ip filter couchdb_in
				ssh $host3 sudo nft delete chain ip filter couchdb_out
}

trap clean exit

hosts=~/.oarnodes
cat $hosts
host1=$(head -1 $hosts)
host2=$(head -2 $hosts | tail -n 1)
host3=$(head -3 $hosts | tail -n 1)
host4=$(head -4 $hosts | tail -n 1)

LOCATIONS_1="$host1 & $host2 & $host3"
LOCATIONS_2="$host3 & $host2 & $host1"

id=$(head -c 20 /dev/urandom | base32)
echo id is $id

lowerid=$(echo $id | tr '[A-Z]' '[a-z'])
v1=$(mktemp kube-v1-XXX.yml)
sed -e "s/kubernetes-bootcamp/deployment-$lowerid/" kube-deploy-v1.yml > $v1

v2=$(mktemp kube-v2-XXX.yml)
sed -e "s/kubernetes-bootcamp/deployment-$lowerid/" kube-deploy-v2.yml > $v2

cleanfiles() {
				rm $v1 $v2
}
trap cleanfiles exit

ssh $host3 'sudo nft delete chain ip filter couchdb_in 2> /dev/null'
ssh $host3 'sudo nft delete chain ip filter couchdb_out 2> /dev/null'

ssh $host3 'sudo nft add chain ip filter couchdb_in "{type filter hook input priority 1; }"'
ssh $host3 sudo nft add rule ip filter couchdb_in ip saddr 127.0.0.1 accept
ssh $host3 sudo nft add rule ip filter couchdb_in tcp dport 5984 drop
ssh $host3 'sudo nft add chain ip filter couchdb_out "{type filter hook output priority 1; }"'
ssh $host3 sudo nft add rule ip filter couchdb_out ip daddr 127.0.0.1 accept
ssh $host3 sudo nft add rule ip filter couchdb_out tcp dport 5984 drop

../../cli/cli exec --id $id --sites "$LOCATIONS_1" --command "sudo kubectl apply -f {$v1}" --type 3
../../cli/cli exec --id $id --sites "$LOCATIONS_2" --command "sudo kubectl apply -f {$v2}" --type 3

read -p "Continue ? "

ssh $host3 sudo nft delete chain ip filter couchdb_in
ssh $host3 sudo nft delete chain ip filter couchdb_out

echo "Waiting for operations to synchronize and run"

python synchronization.py

./run_everywhere.sh "sudo kubectl get deployment deployment-$lowerid"
