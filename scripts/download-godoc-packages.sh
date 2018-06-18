#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

curl -v -sSL -o "packages.$(date +%Y%m%d)" "https://api.godoc.org/packages"

