#!/usr/bin/env bash

. /etc/profile

packages=''

for n in {1..10} ; do
    json="$( \
        curl "https://api.github.com/search/repositories?q=language:go&sort=updated&order=desc&page=${n}&per_page=100" \
        -s \
        -S \
        -H 'accept: application/json' \
        --compressed \
    )"
    additional="$(echo "${json}" | jq -r '.items | .[] | "github.com/" + .full_name')"
    packages="${packages}"$'\n'"${additional}"
done

packages="$(echo "${packages}" | sort | uniq)"

set -x

if [ -n "${packages}" ] ; then
    echo "${packages}"

    #andromeda remote enqueue -a localhost:8001 -v -f $(echo "${packages}")
    andromeda remote enqueue -a 127.0.0.1:8001 -v $(echo "${packages}" | tr $'\n' ' ')
fi

