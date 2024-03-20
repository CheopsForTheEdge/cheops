#!/usr/bin/env sh

. ./env.sh

parallel --nonall --tag --sshloginfile ~/.oarnodes $@
