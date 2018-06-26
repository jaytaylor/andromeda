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

- [ ] Add attribute "CanGoGet" to indicate if package is buildable via `go get`.  Then provide a search filter to only include such packages.
- [ ] Add git version check to crawler (because it's easy to forget to upgrade git!)
- [ ] Add a monitor and require that the disk where the DB is stored always has at least X GB free, where X is based on a multiple of the Bolt database file.  This is to ensure safety that things don't get into a state where data cannot be written to the DB or even worse it gets corrupt.  Remember that DB size may grow non-linearly (need to double check this, but this is what I recall observing).
- [ ] Fix `-s` strangeness, should only specify the base path and auto-append "/src".
- [ ] Handle relative imports.
- [ ] Add analysis of RepoRoot sub-package paths and import names.
- [ ] Remote enqueue

To locate additional TODOs just `find . -name '*.go' -exec grep 'TODO'`

Some of them are only noted in the relevant code region :)

## Development

### Requirements

* [protoc](https://github.com/google/protobuf/releases)

```bash
go get -u github.com/golang/protobuf/...
go get -u github.com/gogo/protobuf/...
go get -u github.com/gogo/gateway/...
go get -u github.com/gogo/googleapis/...
go get -u github.com/grpc-ecosystem/go-grpc-middleware/...
go get -u github.com/grpc-ecosystem/grpc-gateway/...
go get -u github.com/mwitkow/go-proto-validators/...

go generate ./...
```

### License

TBD

