#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ -z "${1:-}" ] ; then
    echo 'ERROR: missing required arguemnt: [andromeda-sub-command]' 1>&2
    exit 1
fi

set -x

subCommand="$1"

# shellcheck disable=SC2009
ps -ef | grep andromeda | grep "${subCommand}" | grep -v grep | awk '{ print $2 }' | xargs -n1 kill -9

