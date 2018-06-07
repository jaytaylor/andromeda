# Dev help

This directory uses protobufs.

The go sourcecode is generated from the protobuf definitions in [package.proto](package.proto).

## Requirements

* [protoc](https://github.com/google/protobuf/releases) installed and available on $PATH
* `go get -u github.com/gogo/protobuf/...`
* `go get -u github.com/gogo/protobuf/protoc-gen-gofast`

## HOWTO Generate go code from protobufs

```bash
cd domain
./gen.sh
protoc \
    --proto_path=. \
    --proto_path=$GOPATH/src \
    --proto_path=$GOPATH/src/github.com/gogo/protobuf/protobuf \
    --gofast_out=plugins=grpc:.,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:. \
    *.proto
```

For additional examples and information, see the gogo repo.  Especially the tests, e.g. [stdtypes/stdtypes.proto](https://github.com/gogo/protobuf/blob/master/test/stdtypes/stdtypes.proto).

