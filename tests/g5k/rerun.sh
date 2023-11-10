#!/usr/bin/env sh

cd $(dirname $0)

. ./env.sh

rm ~/repos/cheops/cli/cli 2> /dev/null

env | grep "_NODE_" | cut -d '=' -f 2 | parallel --tag \
				'rsync --rsync-path="sudo -Sv && rsync" -az --delete ~/repos/cheops {}:/tmp && echo transfer done || echo transfer failed'

parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh /tmp/cheops/tests/g5k/restart.sh

scp $(sed 1q ~/.oarnodes):/tmp/cheops/cli/cli ~/repos/cheops/cli/ > /dev/null

