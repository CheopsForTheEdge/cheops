#!/usr/bin/env sh

cd $(dirname $0)/../..

mkdir -p /tmp/cheops 2> /dev/null
rm -r /tmp/cheops/* 2>/dev/null

echo -n "now at "
git log -1 --oneline

/usr/lib/go-1.19/bin/go build -o /tmp/cheops/cheops.com
killall cheops.com 2> /dev/null
kubectl delete all --all > /dev/null 2>&1
curl -s -XDELETE 'admin:password@localhost:5984/cheops' > /dev/null
curl -s -XDELETE 'admin:password@localhost:5984/_replicator' > /dev/null
curl -s -X PUT http://admin:password@localhost:5984/_replicator > /dev/null

echo "MYFQDN=$(uname -n)" > /tmp/cheops/runenv

rsync -a --delete chephren-ui /tmp/cheops/

for service in cheops chephren
do
				cp $service.service /lib/systemd/system
				systemctl daemon-reload
				systemctl enable $service
				systemctl restart $service
done

status=$(systemctl show cheops | grep ExecMainStatus | cut -d '=' -f 2)
if [ "$status" != "0" ]; then
				echo "startup failed"
				systemctl status cheops
else
				echo "startup done"
fi

cd cli
/usr/lib/go-1.19/bin/go build -o /tmp/cheops/cli/cli
cd ..
