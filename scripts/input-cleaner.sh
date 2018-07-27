#!/usr/bin/env bash

##
# Accepts a newline-delimited list of candidate packages.
#
# When the hostname matches a provider with a standard repository structure, the
# paths are trimmed and de-duped.
#
# Packages with excessive quantities of duplicate strings in the path components
# are filtered out.
#

set -o errexit
set -o pipefail
set -o nounset

if [ "${1:-}" = '-v' ] ; then
    set -x
    shift
fi

file="${1:-/dev/stdin}"
if [ -z "${1:-}" ] ; then
    echo 'INFO: Reading from STDIN' 1>&2
fi

grep -v '\/\(_\?vendor\|Godeps\/_workspace\/src\)\/' < "${file}" \
    | grep -v '^[^\.]*$' \
    | sed 's/^\(\(github\.com\|bitbucket\.com\|bitbucket\.org\|code\.cloudfoundry\.org\|launchpad\.net\|k8s\.io\|gopkg\.in\|jaytaylor\.com\|gigawatt\.io\)\/\([^\/]\+\/[^\/]\+\)\).*/\1/' \
    | sort \
    | uniq \
    | awk '{ split(x, C); n = split($1, F, "/"); n -= 1; m = n; for (i in F) if (i > 0) n -= (C[F[i]]++ > 0); $2 = n; $3 = m; if (m != 0) $4 = n / m ; else $4 = 1 }1' \
    | awk '{ if ($4 >= 0.5) print $1 }'

    #| awk '{ split(x, C); n = split($1, F, "/"); m = n; for (i in F) n -= (C[F[i]]++ > 0); $2 = n / m }1' \
