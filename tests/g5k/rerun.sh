#!/usr/bin/env sh

. ./env.sh

parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh $HOME/repos/cheops/tests/g5k/restart.sh

for bin in cli explorer
do
				scp $(sed 1q ~/.oarnodes):/tmp/cheops/$bin/$bin $(dirname $0)/../../$bin/ > /dev/null
done
