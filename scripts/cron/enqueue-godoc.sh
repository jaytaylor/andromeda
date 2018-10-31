#!/usr/bin/env bash

#
# Description:
#
#     Enqueues packages from godoc.org archive files.
#

set -o errexit
set -o pipefail
set -o nounset

#
# <Configuration>
#

priority=7
serverAddr='127.0.0.1:8001'

#
# </Configuration>
#

ymd="$(date +%Y%m%d)"

infile="${1:-$(dirname "$0")/../../archive.godoc.org/packages.${ymd}}"

if ! [ -e "${infile}" ] && [ -e "${infile}.xz" ] ; then
    xz --decompress --keep "${infile}.xz"
fi

if ! [ -e "${infile}" ] ; then
    echo "ERROR: input file '${infile}' not found" 1>&2
    exit 1
fi

jq -r '.results[].path' < "${infile}" \
    | grep '\.' \
    | xargs \
        andromeda remote enqueue --only-if-not-exists --priority "${priority}" -a "${serverAddr}"

if [ -e "${infile}.xz" ] && ! [[ "${PRESERVE:-}" =~ 1|true|yes ]] ; then
    rm -f "${infile}"
fi

