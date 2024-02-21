#!/usr/bin/env sh

. ./env.sh

parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh $HOME/repos/cheops/tests/g5k/restart.sh

scp $(sed 1q ~/.oarnodes):/tmp/cheops/cli/cli $(dirname $0)/../../cli/ > /dev/null
