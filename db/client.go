package db

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/universe/domain"
)

const (
	TableMetadata = "universe-metadata"
	TablePackages = "packages"
	TableToCrawl  = "to-crawl"
)

var (
	ErrKeyNotFound = errors.New("requested key not found")
)

type DBClient interface {
	Open() error                                                   // Open / start DB client connection.
	Close() error                                                  // Close / shutdown the DB client connection.
	Purge(tables ...string) error                                  // Reset a DB table.
	PackageSave(pkgs ...*domain.Package) error                     // Performs an upsert merge operation on a fully crawled package.
	PackageDelete(pkgPaths ...string) error                        // Delete a package from the index.  Complete erasure.
	Package(pkgPath string) (*domain.Package, error)               // Retrieve a specific package..
	Packages(func(pkg *domain.Package)) error                      // Iterates over all indexed packages and invokes callback on each.
	PackagesWithBreak(func(pkg *domain.Package) bool) error        // Iterates over packages until callback returns false.
	PackagesLen() (int, error)                                     // Number of packages in index.
	ToCrawlAdd(entries ...*domain.ToCrawlEntry) (int, error)       // Only adds entries which don't already exist.  Returns number of new items added.
	ToCrawlSave(entries ...*domain.ToCrawlEntry) error             // Stores all entries.
	ToCrawlDelete(pkgPaths ...string) error                        // Remove a package from the crawl queue.
	ToCrawl(pkgPath string) (*domain.ToCrawlEntry, error)          // Retrieves a single to-crawl entry.
	ToCrawls(func(entry *domain.ToCrawlEntry)) error               // Iterates over all to-crawl entries and invokes callback on each.
	ToCrawlsWithBreak(func(entry *domain.ToCrawlEntry) bool) error // Iterates over to-crawl entries until callback returns false.
	ToCrawlsLen() (int, error)                                     // Number of packages currently awaiting crawl.
	MetaSave(key string, src interface{}) error                    // Store metadata key/value.  NB: src must be one of raw []byte, string, or proto.Message struct.
	MetaDelete(key string) error                                   // Delete a metadata key.
	Meta(key string, dst interface{}) error                        // Retrieve metadata key and populate into dst.  NB: dst must be one of *[]byte, *string, or proto.Message struct.
}

type DBConfig interface {
	Type() DBType // Configuration type specifier.
}

// NewClient constructs a new DB client based on the passed configuration.
func NewClient(config DBConfig) DBClient {
	typ := config.Type()

	switch typ {
	case Bolt:
		return newBoltDBClient(config.(*BoltDBConfig))

	default:
		panic(fmt.Errorf("no client constructor available for db configuration type: %v", typ))
	}
}

// WithDBClient is a convenience utility which handles DB client construction,
// open, and close..
func WithDBClient(config DBConfig, fn func(dbClient DBClient) error) (err error) {
	dbClient := NewClient(config)

	if err = dbClient.Open(); err != nil {
		err = fmt.Errorf("opening DB client: %s", err)
		return
	}
	defer func() {
		if closeErr := dbClient.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("closing DB client: %s", closeErr)
			} else {
				log.Errorf("Existing error before attempt to close DB client: %s", err)
				log.Errorf("Also encountered problem closing DB client: %s", closeErr)
			}
		}
	}()

	if err = fn(dbClient); err != nil {
		return
	}

	return
}
