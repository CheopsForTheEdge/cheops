#!/usr/bin/env sh

. ./env.sh

cat ~/.oarnodes | while read node
do
				echo $node
				curl -s -m 1 "admin:password@$node:5984/_scheduler/docs" | jq -r '.docs[] | .target' | xargs -I {} printf "\t-> {}\n"
done
