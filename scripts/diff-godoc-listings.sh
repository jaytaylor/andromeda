#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ -z "${1:-}" ] || [ -z "${2:-}" ] ; then
    echo 'ERROR: missing one or more required arguments' 1>&2
    echo "usage: $0 <godoc-listing-file-1> <godoc-listing-file-2>" 1>&2
    exit 1
fi

set -x

xzDecompressCmd='xz --decompress --keep --stdout'
#jqSortFilter=".results | sort_by(.path) | .[] | .path""
jqSortFilter=".results | .[] | .path"

comm -3 \
    <(${xzDecompressCmd} "$1" | jq -r "${jqSortFilter}" | sort) \
    <(${xzDecompressCmd} "$2" | jq -r "${jqSortFilter}" | sort)

