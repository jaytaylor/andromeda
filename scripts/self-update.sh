#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

set -x

#mainBranch='master'
mainBranch='jay/distributed-crawler'

tempBranch="tmp-$(date +%s)"

git checkout -b "${tempBranch}"
git branch -D "${mainBranch}"
git fetch --all
git checkout "${mainBranch}"
git branch -D "${tempBranch}"
go get ./...

