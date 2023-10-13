#!/usr/bin/env sh

. ./env.sh

command="multitail"
while read node; do
	command="$command -l \"ssh $node tail -F /tmp/cheops/cheops.log\""
done < <(printenv | grep "$(id -un)_NODE" | cut -d '=' -f 2 )

eval $command
