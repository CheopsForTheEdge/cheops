#!/usr/bin/env sh

. ./env.sh

command="multitail"
while read node; do
	command="$command -l \"ssh $node sudo journalctl -u cheops.service -f\""
done < ~/.oarnodes

eval $command
