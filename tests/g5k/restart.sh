#!/usr/bin/env sh

if [ ! -d /tmp/cheops/.git ]; then
	mkdir -p /tmp/cheops
	cd /tmp
	git clone https://gitlab.inria.fr/discovery/cheops.git
fi

cd /tmp/cheops
git checkout matthieu
git reset --hard origin/matthieu

/usr/lib/go-1.19/bin/go build
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
kubectl delete all --all

MYFQDN=$(uname -n) ./cheops.com 2> cheops.log &
