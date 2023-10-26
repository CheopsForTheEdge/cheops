#!/usr/bin/env sh

. ./env.sh

env | grep "$(id -un)_NODE_" | cut -d '=' -f 2 | while read node
do
				echo $node
				curl -s -m 1 "admin:password@$node:5984/_scheduler/jobs" | jq -r '.jobs[] | .target' | xargs -I {} printf "\t{}\n"
done
