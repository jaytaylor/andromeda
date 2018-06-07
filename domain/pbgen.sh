#!/usr/bin/env bash

set -o errexit

cd "$(dirname "$0")"

set -x

protoc \
    --proto_path=. \
    --proto_path=$GOPATH/src \
    --proto_path=$GOPATH/src/github.com/gogo/protobuf/protobuf \
    --gofast_out=\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types\
plugins=grpc:. \
    *.proto

