#!/usr/bin/env sh

apt-get update && apt-get install -y curl apt-transport-https gnupg

curl https://couchdb.apache.org/repo/keys.asc | gpg --dearmor > /usr/share/keyrings/couchdb-archive-keyring.gpg
. /etc/os-release
echo "deb [signed-by=/usr/share/keyrings/couchdb-archive-keyring.gpg] https://apache.jfrog.io/artifactory/couchdb-deb/ ${VERSION_CODENAME} main" > /etc/apt/sources.list.d/couchdb.list
apt-get update

COUCHDB_PASSWORD=password
echo "couchdb couchdb/mode select standalone \
couchdb couchdb/mode seen true \
couchdb couchdb/bindaddress string 127.0.0.1 \
couchdb couchdb/bindaddress seen true \
couchdb couchdb/adminpass password ${COUCHDB_PASSWORD} \
couchdb couchdb/adminpass seen true \
couchdb couchdb/adminpass_again password ${COUCHDB_PASSWORD} \
couchdb couchdb/adminpass_again seen true" | debconf-set-selections
DEBIAN_FRONTEND=noninteractive apt-get install -y couchdb

f=$(mktemp)
sed 's/\[couchdb\]/[couchdb]\nsingle_node=true/' /opt/couchdb/etc/local.ini \
				| sed 's/\[chttpd\]/[chttpd]\nbind_address = ::/' \
				> $f
mv $f /opt/couchdb/etc/local.ini

systemctl restart couchdb
