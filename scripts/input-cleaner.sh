#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

set -x

if [ -z "${1:-}" ] ; then
    echo 'ERROR: Missing required argument: input filename' 1>&2
    exit 1
fi

jq -r '.results | sort_by(.path) | .[] | .path ' < "$1" \
    | grep -v '\/\(_\?vendor\|Godeps\/_workspace\/src\)\/' \
    | grep -v '^[^\.]*$' | sed 's/^\(\(github\.com\|bitbucket\.com\|code\.cloudfoundry\.org\|launchpad\.net\|k8s\.io\|gopkg\.in\)\/\([^\/]\+\/[^\/]\+\)\).*/\1/' \
    | sed '/^.\{1000\}./d' \
    | uniq \
    | awk '{ split(x, C); n = split($1, F, "/"); n -= 1; m = n; for (i in F) if (i > 0) n -= (C[F[i]]++ > 0); $2 = n; $3 = m; if (n > 1) $4 = n / m ; else $4 = 1 }1' \
    | awk '{ if ($4 >= 0.5) print $1 }'

    #| awk '{ split(x, C); n = split($1, F, "/"); m = n; for (i in F) n -= (C[F[i]]++ > 0); $2 = n / m }1' \
