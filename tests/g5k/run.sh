#!/usr/bin/env sh

. ./env.sh

parallel --nonall --tag --sshloginfile ~/.oarnodes --transferfile restart.sh sudo sh -x restart.sh
