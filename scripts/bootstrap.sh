#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

"./$(dirname "$0")/input-cleaner.sh" archive.godoc.org/packages.20180706 | andromeda -v bootstrap -g - -f t
