#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

if [ -z "${1:-}" ] ; then
    echo 'ERROR: missing required parameter' 1>&2
    echo "usage: $0 <godoc-packages-listing-file>" 1>&2
    exit 1
fi

here="$(dirname "$0")"

"${here}/godoc.org-packages-list.sh" | "${here}/input-cleaner.sh" "$1" | andromeda -v bootstrap -g - -f t

