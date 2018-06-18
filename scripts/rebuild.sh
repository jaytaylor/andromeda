#!/usr/bin/env

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

go generate ./... && go install ./... && go get ./...

