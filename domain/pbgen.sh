#!/usr/bin/env bash

set -o errexit

cd "$(dirname "$0")"

set -x

mkdir -p ../third_party/OpenAPI

protoc \
    --proto_path=. \
    --proto_path=$GOPATH/src/github.com/gogo/googleapis \
    --proto_path=$GOPATH/src/github.com/gogo/protobuf/protobuf \
    --proto_path=$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway \
    --proto_path=$GOPATH/src \
    --gofast_out=plugins=grpc,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:\
$GOPATH/src/ \
    --grpc-gateway_out=\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:\
$GOPATH/src/ \
    --swagger_out=../third_party/OpenAPI/ \
    --govalidators_out=gogoimport=true,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:\
$GOPATH/src \
    *.proto

# Workaround for https://github.com/grpc-ecosystem/grpc-gateway/issues/229.
#sed -i 's/empty.Empty/types.Empty/g' domain/*.pb.gw.go

