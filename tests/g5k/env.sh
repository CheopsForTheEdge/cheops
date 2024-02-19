#!/usr/bin/env sh

oarstat -J -u | jq -r '.[] | .assigned_network_address | .[]' > ~/.oarnodes

awk -F '.' '
	BEGIN 	{print "["}
	NR != 1	{print ","}
					{printf "{\"name\": \"%s\", \"url\": \"http://%s:8080\"}", $1, $0}
	END 		{print "]"}
' ~/.oarnodes > ~/.oarnodes.json
