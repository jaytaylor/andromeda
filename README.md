# Andromeda

Golang Packages Database

Andromeda analyzes the complete graph of the known Go Universe.

## Requirements

* Golang 1.7 or newer
* Git v2.3 or newer (to avoid interactive prompts interrupting the crawler)
* [go-bindata](https://github.com/jteeuwen/go-bindata)
* stringer `go get -u -a golang.org/x/tools/cmd/stringer`

## Installation

```bash
go get jaytaylor.com/andromeda/...
```

### TODOs

[ ] Add attribute "CanGoGet" to indicate if package is buildable via `go get`.  Then provide a search filter to only include such packages.

## Development

### Requirements

* [protoc](https://github.com/google/protobuf/releases)

```bash
go get -u github.com/golang/protobuf/...
go get -u github.com/gogo/protobuf/...
go get -u github.com/gogo/gateway/...
go get -u github.com/gogo/googleapis/...
go get -u github.com/grpc-ecosystem/go-grpc-middleware/...
go get -u github.com/grpc-ecosystem/grpc-gateway
go get -u github.com/grpc-ecosystem/grpc-gateway/...
go get -u github.com/mwitkow/go-proto-validators/...

go generate ./...
```

### License

TBD

