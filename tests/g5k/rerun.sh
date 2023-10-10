#!/usr/bin/env sh

. ./env.sh

parallel --nonall --tag --sshloginfile ~/.oarnodes --line-buffer --transferfile restart.sh sudo sh restart.sh
