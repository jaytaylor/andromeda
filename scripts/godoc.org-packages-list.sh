#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ "${1:-}" = '-v' ] ; then
    set -x
    shift
fi

file="${1:-/dev/stdin}"

if [ -z "${1:-}" ] ; then
    echo 'INFO: Reading input from STDIN' 1>&2
fi

# Automatically handle .xz compressed input files.
decompressCmd='cat'
if [[ "${file}" =~ \.xz$ ]] ; then
    decompressCmd='xz --decompress --keep --stdout'
fi

${decompressCmd} < "${file}" \
    | jq -r '.results | .[] | .path '

