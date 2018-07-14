#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

status="$(systemctl status andromeda-web | grep 'Active:' | awk '{ print $2 }')"

if [ "${status}" = 'active' ] ; then
    sudo systemctl stop andromeda-web
fi

function cleanup() {
    local rc
    rc=$?
    if [ "${status}" = 'active' ] ; then
        sudo systemctl start andromeda-web
    fi
    exit "${rc}"
}
trap cleanup EXIT

function sigCleanup() {
    set +o errexit
    set +o pipefail
    trap '' EXIT
    false # Sets $?
    cleanup
}
trap sigCleanup INT QUIT TERM
# Further reading about traps: https://unix.stackexchange.com/a/322213/22709

mv andromeda.bolt{,.unoptimized}
andromeda rebuild-db -v -b andromeda.bolt.unoptimized -t andromeda.bolt

