#!/usr/bin/env sh

cd /tmp/cheops

echo -n "now at "
git log -1 --oneline

/usr/lib/go-1.19/bin/go build
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
kubectl delete all --all > /dev/null 2>&1
curl -s -XDELETE 'admin:password@localhost:5984/cheops' > /dev/null
curl -s -XDELETE 'admin:password@localhost:5984/_replicator' > /dev/null
curl -s -X PUT http://admin:password@localhost:5984/_replicator > /dev/null

echo "MYFQDN=$(uname -n)" > runenv

cp cheops.service /lib/systemd/system
systemctl daemon-reload
systemctl enable cheops
systemctl restart cheops

status=$(systemctl show cheops | grep ExecMainStatus | cut -d '=' -f 2)
if [ "$status" != "0" ]; then
				echo "startup failed"
				systemctl status cheops
else
				echo "startup done"
fi

for bin in cli explorer
do
				cd $bin
				/usr/lib/go-1.19/bin/go build
				cd ..
done
