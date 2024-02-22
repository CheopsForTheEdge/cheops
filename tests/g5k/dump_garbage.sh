#!/usr/bin/env sh

. ./env.sh

cat ~/.oarnodes | while read node
do
				echo $node
				curl  -s -XPOST -H 'Content-Type: application/json' "$node:5984/cheops/_find" --data-binary "{\"selector\": {\"Type\": \"DELETE\", \"Locations\": {\"\$elemMatch\": {\"\$eq\": \"$node\"}}}}"	\
								| jq -rc '.docs[] | "id=" + .ResourceId + " rev=" + .ResourceRev' \
								| awk '{print "\t" $0}'
done
