#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")"

mkdir -p ../third_party/OpenAPI

REWRITES=Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor

set -x

protoc \
    --proto_path=. \
    --proto_path="$GOPATH/src/github.com/gogo/googleapis" \
    --proto_path="$GOPATH/src/github.com/gogo/protobuf/protobuf" \
    --proto_path="$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway" \
    --proto_path="$GOPATH/src" \
    --gogofast_out=plugins=grpc,\
"$REWRITES:$GOPATH/src/" \
    --grpc-gateway_out=\
"$REWRITES:$GOPATH/src/" \
    --swagger_out=../third_party/OpenAPI/ \
    --govalidators_out=gogoimport=true,\
"$REWRITES:$GOPATH/src/" \
    --gorm_out="engine=postgres,enums=string:$GOPATH/src/" \
    *.proto

# Workaround for https://github.com/grpc-ecosystem/grpc-gateway/issues/229.
#sed -i 's/empty.Empty/types.Empty/g' domain/*.pb.gw.go

