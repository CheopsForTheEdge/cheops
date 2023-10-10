#!/usr/bin/env sh

if [ ! -d /tmp/cheops/.git ]; then
	mkdir -p /tmp/cheops
	cd /tmp
	git clone https://gitlab.inria.fr/discovery/cheops.git 2>&1
fi

cd /tmp/cheops
git reset --hard > /dev/null 2>&1
git checkout operations > /dev/null 2>&1
git fetch > /dev/null 2>&1
git reset --hard origin/operations

/usr/lib/go-1.19/bin/go build
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
kubectl delete all --all > /dev/null 2>&1

echo starting
MYFQDN=$(uname -n) ./cheops.com 2> cheops.log &
