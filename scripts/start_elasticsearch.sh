#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

sudo docker run \
    --rm \
    -p 9200:9200 \
    -p 9300:9300 \
    --ulimit nofile=65535:65535 \
    -v "${GOPATH}/src/jaytaylor.com/andromeda/scripts/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml" \
    -v /mnt/nvme1/esdata:/usr/share/elasticsearch/data \
    -e 'ES_JAVA_OPTS=-Xms512m -Xmx512m' \
    docker.elastic.co/elasticsearch/elasticsearch:7.0.1 bin/elasticsearch

