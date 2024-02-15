#!/usr/bin/env sh

i=0

while read var
do
				unset $var
done < <(printenv | grep "$(id -un)_NODE")

while read node
do
				i=$(($i + 1))
				var="$(id -un)_NODE_$i"
				printf -v $var $node
				export $var
done < <(oarstat -J -u | jq -r '.[] | .assigned_network_address | .[]' )

oarstat -J -u | jq -r '.[] | .assigned_network_address | .[]' > ~/.oarnodes

awk -F '.' '
	BEGIN 	{print "["}
	NR != 1	{print ","}
					{printf "{\"name\": \"%s\", \"url\": \"http://%s:8080\"}", $1, $0}
	END 		{print "]"}
' ~/.oarnodes > ~/.oarnodes.json
