#!/usr/bin/env sh

hosts=~/.oarnodes
host1=$(head -1 $hosts)
host2=$(head -2 $hosts | tail -n 1)
host3=$(head -3 $hosts | tail -n 1)
host4=$(head -4 $hosts | tail -n 1)

LOCATIONS_1="$host1 & $host2 & $host3"
LOCATIONS_2="$host1 & $host2 & $host4"

echo $LOCATIONS_1

../../cli/cli exec --id left --sites "$LOCATIONS_1" --command 'mkdir -p /tmp/foo; touch /tmp/foo/left'

read -p "Continue ? "

../../cli/cli exec --id right --sites "$LOCATIONS_2" --command 'mkdir -p /tmp/foo; touch /tmp/foo/right'
