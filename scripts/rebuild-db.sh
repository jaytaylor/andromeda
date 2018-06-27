#!/usr/bin/env bash

cd "$(dirname "$0")/.."

sudo systemctl stop andromeda-web

mv andromeda.bolt{,.unoptimized}
andromeda rebuild-db -v -b andromeda.bolt.unoptimized -t andromeda.bolt

sudo systemctl start andromeda-web

