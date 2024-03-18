#!/usr/bin/env sh

. ./env.sh

clean() {
				ssh $host3 sudo nft delete chain inet filter couchdb_in
				ssh $host3 sudo nft delete chain inet filter couchdb_out
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

ssh $host3 'sudo nft delete chain inet filter couchdb_in 2> /dev/null'
ssh $host3 'sudo nft delete chain inet filter couchdb_out 2> /dev/null'

ssh $host3 'sudo nft add chain inet filter couchdb_in "{type filter hook input priority 1; }"'
ssh $host3 sudo nft add rule inet filter couchdb_in ip saddr 127.0.0.1 accept
ssh $host3 sudo nft add rule inet filter couchdb_in tcp dport 5984 drop
ssh $host3 'sudo nft add chain inet filter couchdb_out "{type filter hook output priority 1; }"'
ssh $host3 sudo nft add rule inet filter couchdb_out ip daddr 127.0.0.1 accept
ssh $host3 sudo nft add rule inet filter couchdb_out tcp dport 5984 drop


../../cli/cli exec --id $id --sites "$LOCATIONS_1" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=insert&value=13'" --type 3
../../cli/cli exec --id $id --sites "$LOCATIONS_1" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=add&value=7'" --type 2
../../cli/cli exec --id $id --sites "$LOCATIONS_1" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=add&value=2'" --type 2

../../cli/cli exec --id $id --sites "$LOCATIONS_2" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=insert&value=3'" --type 3
../../cli/cli exec --id $id --sites "$LOCATIONS_2" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=add&value=-4'" --type 2
../../cli/cli exec --id $id --sites "$LOCATIONS_2" --command "curl -s -XPOST 'localhost:9090/$id?type=counter&operation=add&value=-1'" --type 2

parallel --nonall --tag --sshloginfile ~/.oarnodes "curl -s 'localhost:9090/$id?type=counter'; echo"

read -p "Continue ? "

ssh $host3 sudo nft delete chain inet filter couchdb_in
ssh $host3 sudo nft delete chain inet filter couchdb_out

python synchronization.py
parallel --nonall --tag --sshloginfile ~/.oarnodes "curl -s 'localhost:9090/$id?type=counter'; echo"
