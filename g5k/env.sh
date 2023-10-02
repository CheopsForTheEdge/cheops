#!/usr/bin/env sh

i=0

while read node
do
				i=$(($i + 1))
				var="$(id -un)_NODE_$i"
				printf -v $var $node
				export $var
done < <(oarstat -J -u | jq -r '.[] | .assigned_network_address | .[]' )
