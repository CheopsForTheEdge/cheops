#!/usr/bin/env sh

. ./env.sh

env | grep "$(id -un)_NODE_" | cut -d '=' -f 2 | while read node
do
				echo $node
				curl  -s -XPOST -H 'Content-Type: application/json' "$node:5984/cheops/_find" --data-binary "{\"selector\": {\"Locations\": {\"\$not\": {\"\$elemMatch\": {\"\$eq\": \"$node\"}}}}}"	\
								| jq -r '.docs[] | ._id' | while read id; do
								if [ -z "$id" ]
							 	then
												echo "\tNothing"
								else
												echo -e "\thttps://$node:5984/cheops/$id"
								fi
								done
done
