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
./pbgen.sh
```

For additional examples and information, see the gogo repo.  Especially the tests, e.g. [stdtypes/stdtypes.proto](https://github.com/gogo/protobuf/blob/master/test/stdtypes/stdtypes.proto).

