#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/../archive.godoc.org/"

find . -type f -name '*.xz' | sort -r | xargs -n1 ../scripts/bootstrap.sh
