#!/usr/bin/env sh

cd /tmp/cheops
git log -1 --oneline

/usr/lib/go-1.19/bin/go build
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
kubectl delete all --all > /dev/null 2>&1

echo starting
MYFQDN=$(uname -n) ./cheops.com 2> cheops.log &
