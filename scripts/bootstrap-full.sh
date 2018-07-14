#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")/../archive.godoc.org/"

ls -1 | sort -r | xargs -n1 ../scripts/bootstrap.sh
