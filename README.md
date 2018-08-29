# Andromeda

Golang Packages Database

Andromeda analyzes the complete graph of the known Go Universe.

## Requirements

* Golang 1.7 or newer
* Git v2.3 or newer (to avoid interactive prompts interrupting the crawler)
* [go-bindata](https://github.com/jteeuwen/go-bindata)
* stringer `go get -u -a golang.org/x/tools/cmd/stringer`
* OpenSSL (for automatic retrieval of SSL/TLS server public-key to feed gRPC by remote-crawler)
* xz (for downloading daily snapshots from godoc.org)

## Installation

```bash
go get jaytaylor.com/andromeda/...
```

### Backend selection

Additional setup may be required depending on which DB backend you want to use.

#### BoltDB

No additional packages or work necessary.

#### RocksDB

1. Install RocksDB

[RocksDB installation instructions](https://github.com/facebook/rocksdb/blob/master/INSTALL.md).

2. Install the RocksDB golang package

[gorocksdb package installation instructions](https://github.com/tecbot/gorocksdb#install).

3. Build andromeda with the RocksDB backend enabled

```bash
go get jaytaylor.com/andromeda/...
cd "${GOPATH}/src/jaytaylor.com/andromeda
go build -o andromeda -tags rocks
```

#### Postgresql

1. Install postgresql

```bash
apt-get install \
    postgresql \
    postgresql-client \
    postgresql-contrib \
    postgresql-10-prefix
```

2. Enable the prefix module

["prefix" module enablement instructions](https://github.com/dimitri/prefix#installation).

### TODOs

#### Fancy Data

- [ ] Add attribute "CanGoGet" to indicate if package is buildable via `go get`.  Then provide a search filter to only include such packages.
- [ ] Add `deprecated` attribute and set based on detection of phrases like "is deprecated" and "superceded by" in README.
- [ ] Attempt topic extraction from READMEs.
- [ ] Add usage popularity comparison between 2 pkgs.
- [ ] Generic fork and hierarchy detection: Find packages with the pkg same name, then use `lastCommitedAt` field to derive possibly related packages.  Take this list and inspect commit histories to determine if there is commit-overlap between the two.  Could implement github scraping hacks to verify accuracy.
- [ ] Handle relative imports (x2).
- [ ] Detect and persist whether each import is vendored or not in the reverse-imports mapping data.
- [ ] Add analysis of RepoRoot sub-package paths and import names.
- [ ] Add counts for total number of packages tracked, globally (currently repos are tracked and called "packages" everywhere, ugh..).
- [ ] 1/2 Distinguish between pkg and repo by refactoring what is currently called a "package" in andromeda into a "repo".
- [ ] 2/2 Add alias tracking table.

#### Remote Crawlers

- [ ] 1/4 Add a `--id` flag for remote crawlers to uniquely identify them (x2).
- [ ] 2/4 Remote crawlers should track and store their own statistics in a local bolt db file, per crawler-id.  For example, keep track of number of crawls done per day, total size of crawled content, number of successful and failed crawls.
- [ ] 3/4 Server-side: Track crawlers by ID, and track when they were last seen, IP addresses, number of packages crawled, number of successful crawls vs errors.
- [ ] 4/4 Provide live-query mechanism for server to ping all crawlers to get an accurate count of actives.  Would also be interesting to have the crawlers include their version (git hash) and crawl stats in the response.

#### Fully Autonomous System

- [ ] Add queue monitor, when it is empty add N least recently updated packages to crawl.
- [X] Add errors counter to ToCrawlEntry and throw away when error count exceeds N.
- [ ] Add process-level concurrency support for remote crawlers (to increase throughput without resorting to trying to manage multiple crawler processes per host).

#### Data Integrity

- [ ] To avoid dropping items across restarts, implement some kind of a WAL and resume functionality (x2, see next item below).
- [ ] Protect against losing queue items from process restarts / interruptions; Add in-flight TCE's to an intermediate table, and at startup move items from said table back into the to-crawl queue.
- [ ] Remote-crawler: Store crawl result on disk when sending failed, then when remote starts, check for failed transmit and send it.  Possible complexity due to server not expecting _that_ crawl result.  May need to expose via different gRPC API endpoint than `Attach`.

#### Operational and Performance

- [ ] Fix `-s` strangeness, should only specify the base path and auto-append "/src".
- [ ] Consider refactoring "Package" to "Repo", since a go repo contains an arbitrary number of packages (I known, "yuck", but..).
- [X] Implement Postgres backend.
- [X] Implement Postgres queue.
- [ ] Implement CockroachDB backend.
- [ ] Implement CockroachDB queue.
- [ ] Add git commit hash to builds, and has gRPC client send it with requests.
- [ ] Implement pure-postgres native db.Client interface and see if or how much better we can do compared to K/V approach.
- [X] Implement pending references updates as a batch job (currently it's been disabled due to low performance).  Another way to solve it would be to only save pending references sometimes - just add an extra parameter on the internal save method (went with this, was very simple to add a single param to the save functions to avoid merging pending references for recursively-triggered saves.

#### Uncategorized

- [X] Add git version check to crawler (because it's easy to forget to upgrade git!).  Note: This is part of the `check` command, also verifies availability of openssl binary.
- [ ] Add a monitor and require that the disk where the DB is stored always has at least X GB free, where X is based on a multiple of the Bolt database file.  This is to ensure safety that things don't get into a state where data cannot be written to the DB or even worse it gets corrupt.  Remember that DB size may grow non-linearly (need to double check this, but this is what I recall observing).
- [ ] Move failed to-crawls to different table instead of dropping them outright.
- [ ] 1/2 Expose queue contents over rest API.
- [ ] 2/2 Frontend viewer for queue head and tail contents.

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

### How to bootstrap the server

#### Example

Grab latest seed list from godoc.org:

```
./download-godoc-packages.sh
```

Locate the downloaded file and extract it with `xz -k -d <filename>`.

Then cleanup the input and seed into andromeda:

```
./scripts/input-cleaner.sh archive.godoc.org/packages.20180706 \
    | andromeda bootstrap -g - -f text
```

### Installation instructions

Instructions generally live alongside the code within the header of the relevant program, so always check the top of the scripts and source code for installation instructions and per-script documentation.

The exception to this rule is the andromeda binary, where usage instructions are available by running `andromeda --help` or `andromeda <sub-command> --help`.

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
### Running remote-crawler behind a proxy

#### Linux
Host github.com gitlab.com bitbucket.com bitbucket.org code.cloudfoundry.org launchpad.net
    ProxyCommand ncat --proxy proxy.example.com:80 %h %p
    Compression yes

##### macOS
Host github.com gitlab.com bitbucket.com bitbucket.org code.cloudfoundry.org launchpad.net
    ProxyCommand nc -X connect -x proxy.example.com:80 %h %p
    Compression yes

### Commands

#### Add top 1000 most recently committed packages to the crawl queue

```bash
cp -a andromeda.bolt a
andromeda -b a -v stats mru -n 1000 | jq -r '.[] | .path' | xargs -n10 andromeda remote enqueue -a 127.0.0.1:8001 -v -f
rm a
```

#### Cross-datastore migrations

##### Migrate from postgres to bolt

```bash
andromeda util rebuild-db \
    -v \
    --driver postgres \
    --db "dbname=andromeda host=/var/run/postgresql" \
    --rebuild-db-driver bolt \
    --rebuild-db-file new.bolt \
```

##### Migrate from bolt to postgres, filtering out package histories

```bash
andromeda util rebuild-db \
    -v \
    --driver bolt \
    --db no-history.bolt \
    --rebuild-db-driver postgres \
    --rebuild-db-file "dbname=andromeda host=/var/run/postgresql" \
    --rebuild-db-filters clearHistories
```

### License

TBD

