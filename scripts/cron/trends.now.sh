#!/usr/bin/env bash

. /etc/profile

packages=''

for n in {1..10} ; do
    json="$( \
        curl 'https://trends.now.sh/graphql' \
        -s \
        -S \
        -H 'content-type: application/json' \
        -H 'accept: application/json' \
        --data-binary '{"query":"query TopGolang {repos(language: \"go\", time: 8) {full_name}}","variables":null,"operationName":"TopGolang"}' \
        --compressed \
    )"
    additional="$(echo "${json}" | jq -r '.data.repos | .[] | "github.com/" + .full_name')"
    packages="${packages}"$'\n'"${additional}"
done

packages="$(echo "${packages}" | sort | uniq)"

if [ -n "${packages}" ] ; then
    echo "${packages}"

    #andromeda remote enqueue -a localhost:8001 -v -f $(echo "${packages}")
    andromeda remote enqueue -a 127.0.0.1:8001 -v $(echo "${packages}" | tr $'\n' ' ')
fi
