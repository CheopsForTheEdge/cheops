#!/usr/bin/env sh

before=$(pwd)
trap "cd $before" EXIT

cd $(dirname $0)/../..
here=$(pwd)

. tests/g5k/env.sh

rm $here/cli/cli 2> /dev/null

cat ~/.oarnodes | parallel --tag \
				"rsync --rsync-path='sudo -Sv && rsync' -az --delete $here/ {}:/tmp/cheops && echo transfer done || echo transfer failed"

parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh /tmp/cheops/tests/g5k/restart.sh

for bin in cli explorer
do
				scp $(sed 1q ~/.oarnodes):/tmp/cheops/$bin/$bin $here/$bin/ > /dev/null
done
