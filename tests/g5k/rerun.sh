#!/usr/bin/env sh

. ./env.sh

here=$(dirname $(realpath $0))
parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh $here/restart.sh

scp $(sed 1q ~/.oarnodes):/tmp/cheops/cli/cli $(dirname $0)/../../cli/ > /dev/null
