#!/usr/bin/env sh

. ./env.sh

command="multitail"
while read node; do
	command="$command -l \"ssh $node sudo journalctl -u cheops.service -f\""
done < <(printenv | grep "$(id -un)_NODE" | sort | cut -d '=' -f 2 )

eval $command
