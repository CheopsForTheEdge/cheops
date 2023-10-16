#!/usr/bin/env sh

cd /tmp/cheops

echo -n "now at "
git log -1 --oneline

/usr/lib/go-1.19/bin/go build
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
kubectl delete all --all > /dev/null 2>&1

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
