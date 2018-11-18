#!/usr/bin/env bash

##
#
# Andromeda self-update script for crawler instances.
#
# See crawler-service-udpate.sh for installation instructions.
#
#
# Exit status code map:
#
#     0: Update found and client rebuilt successfully.
#
#     100: No update found.
#
#
# Other exit status codes are also possible, for example if a `git` or `go get`
# command fails.
#
##

set -o errexit
set -o pipefail

source /etc/profile
# N.B. /etc/profile frequently seems to contain unbound variable references.
set -o nounset

if [ "${DEBUG:-0}" -ne 0 ] ; then
    echo 'DEBUG: Enabling -x mode for verbose script debug output' 2>&1
    set -x
fi

cd "$(dirname "$0")/.."

mainBranch='master'

oldCommitHash="$(git rev-parse HEAD)"

tempBranch="tmp-$(date +%s)"

git stash || :
git checkout -b "${tempBranch}"
git branch -D "${mainBranch}"
git fetch --all
git checkout "${mainBranch}"
git pull origin "${mainBranch}"
git branch -D "${tempBranch}"

newCommitHash="$(git rev-parse HEAD)"

if [ "${newCommitHash}" = "${oldCommitHash}" ] ; then
    echo 'DEBUG: No updates available at this time' 2>&1
    exit 100
fi

go get ./...

