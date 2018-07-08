# Andromeda

Golang Packages Database

Andromeda analyzes the complete graph of the known Go Universe.

## Requirements

* Golang 1.7 or newer
* Git v2.3 or newer (to avoid interactive prompts interrupting the crawler)
* [go-bindata](https://github.com/jteeuwen/go-bindata)
* stringer `go get -u -a golang.org/x/tools/cmd/stringer`
* OpenSSL (for automatic retrieval of SSL/TLS server public-key to feed gRPC by remote-crawler)

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
- [ ] Detect and persist whether each import is vendored or not in the reverse-imports mapping data.

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

### Running remote-crawler as a system service on Windows

Ensure the target user account has the "Run as a System Service" Policy.

Perform the following to edit the Local Security Policy of the computer you want to define the 'Logon as a Service' permission:

1.Logon to the computer with administrative privileges. 
2.Open the 'Administrative Tools' and open the 'Local Security Policy'.
3.Expand 'Local Policy' and click on 'User Rights Assignment'.
4.In the right pane, right-click 'Log on as a service' and select properties. 
5.Click on the 'Add User or Group' button to add the new user. 
6.In the 'Select Users or Groups' dialogue, find the user you wish to enter and click 'OK'.
7.Click 'OK' in the 'Log on as a service Properties' to save changes. 

Notes:

Ensure that the user which you have added above is not listed in the 'Deny log on as a service' policy in the Local Security Policy.

#### Example system service installation on windows

```bash
andromeda service crawler install -v --delete-after -s /tmp/src -a <host.name>:443 -c <path-to-letsencrypt-cert.pem> -u .\<windows-username> -p <windows-password>
```

### License

TBD

