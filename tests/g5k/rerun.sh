#!/usr/bin/env sh

. ./env.sh

# from https://stackoverflow.com/a/246128, get the repository where the executed script is located
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer sudo sh $SCRIPT_DIR/restart.sh

scp $(sed 1q ~/.oarnodes):/tmp/cheops/cli/cli $(dirname $0)/../../cli/ > /dev/null
