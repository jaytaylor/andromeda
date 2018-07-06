package db

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/domain"
)

const (
	TableMetadata          = "andromeda-metadata"
	TablePackages          = "packages"
	TablePendingReferences = "pending-references"
	TableToCrawl           = "to-crawl"

	DefaultQueuePriority = 3
)

var (
	ErrKeyNotFound    = errors.New("requested key not found")
	ErrNotImplemented = errors.New("function not implemented")
)

type Client interface {
	Open() error                                                                                   // Open / start DB client connection.
	Close() error                                                                                  // Close / shutdown the DB client connection.
	Purge(tables ...string) error                                                                  // Reset a DB table.
	EachRow(table string, fn func(k []byte, v []byte)) error                                       // Invoke a callback on the key/value pair for each row of the named table.
	EachRowWithBreak(table string, fn func(k []byte, v []byte) bool) error                         // Invoke a callback on the key/value pair for each row of the named table until cb returns false.
	PackageSave(pkgs ...*domain.Package) error                                                     // Performs an upsert merge operation on a fully crawled package.
	PackageDelete(pkgPaths ...string) error                                                        // Delete a package from the index.  Complete erasure.
	Package(pkgPath string) (*domain.Package, error)                                               // Retrieve a specific package..
	Packages(pkgPaths ...string) (map[string]*domain.Package, error)                               // Retrieve several packages.
	EachPackage(func(pkg *domain.Package)) error                                                   // Iterates over all indexed packages and invokes callback on each.
	EachPackageWithBreak(func(pkg *domain.Package) bool) error                                     // Iterates over packages until callback returns false.
	PathPrefixSearch(prefix string) (map[string]*domain.Package, error)                            // Search for packages with paths matching a specific prefix.
	PackagesLen() (int, error)                                                                     // Number of packages in index.
	RecordImportedBy(refPkg *domain.Package, resources map[string]*domain.PackageReferences) error // Save imported-by relationship updates.
	ToCrawlAdd(entries []*domain.ToCrawlEntry, opts *QueueOptions) (int, error)                    // Only adds entries which don't already exist.  Returns number of new items added.
	ToCrawlRemove(pkgs []string) (int, error)                                                      // Scrubs items from queue.
	ToCrawlDequeue() (*domain.ToCrawlEntry, error)                                                 // Pop an entry from the crawl queue.
	EachToCrawl(func(entry *domain.ToCrawlEntry)) error                                            // Iterates over all to-crawl entries and invokes callback on each.
	EachToCrawlWithBreak(func(entry *domain.ToCrawlEntry) bool) error                              // Iterates over to-crawl entries until callback returns false.
	ToCrawlsLen() (int, error)                                                                     // Number of packages currently awaiting crawl.
	MetaSave(key string, src interface{}) error                                                    // Store metadata key/value.  NB: src must be one of raw []byte, string, or proto.Message struct.
	MetaDelete(key string) error                                                                   // Delete a metadata key.
	Meta(key string, dst interface{}) error                                                        // Retrieve metadata key and populate into dst.  NB: dst must be one of *[]byte, *string, or proto.Message struct.
	BackupTo(destFile string) error                                                                // Create a backup copy of the DB.  Return ErrNotImplemented if not supported.
	RebuildTo(newDBFile string, kvFilters ...KeyValueFilterFunc) error                             // Rebuild a fresh copy of the DB at destination.  Return ErrNotImplmented if not supported.  Optionally pass in one or more KeyValueFilterFunc functions.
	PendingReferences(pkgPathPrefix string) ([]*domain.PendingReferences, error)                   // Retrieve pending references listing for a package path prefix.
	PendingReferencesSave(pendingRefs ...*domain.PendingReferences) error                          // Save pending references.
	PendingReferencesDelete(keys ...string) error                                                  // Delete pending references keys.
	PendingReferencesLen() (int, error)                                                            // Number of pending references keys.
}

type Config interface {
	Type() Type // Configuration type specifier.
}

// NewClient constructs a new DB client based on the passed configuration.
func NewClient(config Config) Client {
	typ := config.Type()

	switch typ {
	case Bolt:
		return newBoltDBClient(config.(*BoltConfig))

	default:
		panic(fmt.Errorf("no client constructor available for db configuration type: %v", typ))
	}
}

type QueueOptions struct {
	Priority int
}

func NewQueueOptions() *QueueOptions {
	opts := &QueueOptions{
		Priority: DefaultQueuePriority,
	}
	return opts
}

type KeyValueFilterFunc func(bucket []byte, key []byte, value []byte) (keyOut []byte, valueOut []byte)

// WithClient is a convenience utility which handles DB client construction,
// open, and close..
func WithClient(config Config, fn func(dbClient Client) error) (err error) {
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
