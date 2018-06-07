package db

import (
	"errors"
	"fmt"

	"github.com/jaytaylor/universe/domain"
)

var (
	ErrKeyNotFound = errors.New("requested key not found")
)

type DBClient interface {
	Open() error                                          // Open / start DB client connection.
	Close() error                                         // Close / shutdown the DB client connection.
	PackageSave(pkgs ...*domain.Package) error            // Performs an upsert merge operation on a fully crawled package.
	PackageDelete(pkgNames ...string) error               // Delete a package from the index.  Complete erasure.
	Package(pkgName string) (*domain.Package, error)      // Retrieve a specific package..
	Packages(func(pkg *domain.Package)) error             // Iterates over all indexed packages and invokes callback on each.
	PackagesLen() (int, error)                            // Number of packages in index.
	ToCrawlAdd(entries ...*domain.ToCrawlEntry) error     // reason should	indicate where / how the request to queue originated.
	ToCrawlDelete(pkgNames ...string) error               // Remove a package from the crawl queue.
	ToCrawl(pkgName string) (*domain.ToCrawlEntry, error) // Retrieves a single to-crawl entry.
	ToCrawls(func(entry *domain.ToCrawlEntry)) error      // Iterates over all to-crawl entries and invokes callback on each.
	ToCrawlsLen() (int, error)                            // Number of packages currently awaiting crawl.
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
