#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

go generate ./domain/... ./web/... && go install ./... && go get ./...
