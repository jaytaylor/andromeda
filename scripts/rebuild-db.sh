#!/usr/bin/env bash
set -x


cd "$(dirname "$0")/.."

status="$(systemctl status andromeda-web | (grep 'Active:' || :) | awk '{ print $2 }')"
set -o errexit
set -o pipefail
set -o nounset
was_active=false

if [ "${status}" = 'active' ] ; then
    was_active=true
    sudo systemctl stop andromeda-web
fi

function cleanup() {
    local rc
    rc=$?
    if [ "${was_active}" = true ] && [ "${status}" = 'active' ] ; then
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

