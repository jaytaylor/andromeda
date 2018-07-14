#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ -z "${1:-}" ] || [ -z "${2:-}" ] ; then
    echo 'ERROR: missing one or more required arguments' 1>&2
    echo "usage: $0 <godoc-listings-dir> <report-output-dir>" 1>&2
    exit 1
fi

if ! [ -d "$1" ] ; then
    echo 'ERROR: godoc-listings-dir must be a directory' 1>&2
    echo "usage: $0 <godoc-listings-dir> <report-output-dir>" 1>&2
    exit 1
fi

listingsDir="$1"
outputDir="$2"

if ! [ -d "${outputDir}" ] ; then
    echo "DEBUG: creating output directory: ${outputDir}"
    mkdir -p "${outputDir}"
fi

#set -x

differ="$(dirname "$(realpath "$0")")/diff-godoc-listings.sh"

a=''
b=''

# Iterate over all a-b file pairs.
# `a` is always the older one.
for f in "${listingsDir}"/*.xz ; do
    if [ -z "${a}" ] ; then
        a="${f}"
        continue
    fi
    b="${f}"
    # shellcheck disable=SC2001
    report="$(echo "${a}" | sed 's/^.*packages\.\(.*\)\.xz$/\1/')-$(echo "${b}" | sed 's/^.*packages\.\(.*\)\.xz$/\1/').txt"
    "${differ}" "${a}" "${b}" > "${outputDir}/${report}"
    a="${f}"
done

