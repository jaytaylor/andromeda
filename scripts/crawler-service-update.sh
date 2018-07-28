#!/usr/bin/env bash

##
#
# Andromeda Crawler service updater.
#
# Known to be compatible with Debian/Ubuntu and RedHat -flavored linux
# distributions.
#
# Installation:
#
#     * Add to root crontab:
#
#         */15 * * * * <GOPATH-HERE>/src/jaytaylor.com/andromeda/scripts/crawler-service-udpate.sh 1>/dev/null 2>/dev/null
#
##

function restartCrawlerService() {
    local rc

    echo 'INFO: Restarting the Andromeda Crawler system service' 1>&2

    set +o errexit
    set +o pipefail
    # if $(command -v systemctl 1>/dev/null 2>/dev/null) ; then
    command -v systemctl 1>/dev/null 2>/dev/null
    rc=$?
    if [ "${rc}" -eq 0 ] ; then
        /bin/systemctl restart andromeda-crawler
        rc=$?
        /bin/systemctl status andromeda-crawler
    else
        /usr/sbin/service andromeda-crawler restart
        rc=$?
        /usr/sbin/service andromeda-crawler status
    fi
    set -o errexit
    set -o pipefail

    return "${rc}"
}
export -f restartCrawlerService

function main() {
    set -o errexit
    set -o pipefail

    local rc
    local owner

    if [ "$(id -u)" -ne 0 ] ; then
        echo "ERROR: $0 must be run as root" 1>&2
        exit 1
    fi

    if [ "${DEBUG:-0}" -ne 0 ] ; then
        echo 'DEBUG: Enabling -x mode for verbose script debug output' 2>&1
        set -x
    fi

    source /etc/profile
    # N.B. /etc/profile frequently seems to contain unbound variable references.
    set -o nounset

    cd "$(dirname "$0")"

    owner="$(stat --format '%U' self-update.sh)"

    set +o errexit
    set +o pipefail
    sudo -u "${owner}" ./self-update.sh
    rc=$?

    echo "DEBUG: self-update.sh exited with status code=${rc}" 2>&1

    if [ "${rc}" -eq 0 ] ; then
        restartCrawlerService
        rc=$?
    elif [ "${rc}" -eq 100 ] ; then
        return 0
    fi
    return "${rc}"
}
export -f main

if [ "${BASH_SOURCE[0]}" = "${0}" ] ; then
    # Only auto-run when being executed (and don't auto-run functions when being sourced).
    main
fi

