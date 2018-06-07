The go sourcecode is generated from the protobuf definitions in [package.proto](package.proto).

Generation command:

```bash
protoc \
    --proto_path=. \
    --proto_path=resources/pb/include/ \
    --go_out=plugins=grpc:. \
    *.proto
```

