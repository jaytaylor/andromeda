#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/.."

mkdir -p archive.godoc.org
cd archive.godoc.org

filename="packages.$(date -u +%Y%m%d)"

curl -v -sSL -o "${filename}" "https://api.godoc.org/packages"

xz --compress --extreme "${filename}"

