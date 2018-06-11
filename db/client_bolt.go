package db

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/bbolt"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/universe/domain"
)

var (
	ErrMetadataUnsupportedSrcType = errors.New("unsupported src type: must be an []byte, string, or proto.Message")
	ErrMetadataUnsupportedDstType = errors.New("unsupported dst type: must be an *[]byte, *string, or proto.Message")
)

type BoltDBConfig struct {
	DBFile      string
	BoltOptions *bolt.Options
}

func NewBoltDBConfig(dbFilename string) *BoltDBConfig {
	cfg := &BoltDBConfig{
		DBFile: dbFilename,
		BoltOptions: &bolt.Options{
			Timeout: 1 * time.Second,
		},
	}
	return cfg
}

func (cfg BoltDBConfig) Type() DBType {
	return Bolt
}

type BoltDBClient struct {
	config *BoltDBConfig
	db     *bolt.DB
	mu     sync.Mutex
}

func newBoltDBClient(config *BoltDBConfig) *BoltDBClient {
	client := &BoltDBClient{
		config: config,
	}
	return client
}

func (client *BoltDBClient) Open() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.db != nil {
		return nil
	}

	db, err := bolt.Open(client.config.DBFile, 0600, client.config.BoltOptions)
	if err != nil {
		return err
	}
	client.db = db

	if err := client.initDB(); err != nil {
		return err
	}

	return nil
}

func (client *BoltDBClient) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.db == nil {
		return nil
	}

	if err := client.db.Close(); err != nil {
		return err
	}

	client.db = nil

	return nil
}

func (client *BoltDBClient) initDB() error {
	return client.db.Update(func(tx *bolt.Tx) error {
		buckets := []string{
			TableMetadata,
			TablePackages,
			TableToCrawl,
		}
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("initDB: creating bucket %q: %s", name, err)
			}
		}
		return nil
	})
}

func (client *BoltDBClient) Purge(tables ...string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		for _, table := range tables {
			log.WithField("bucket", table).Debug("dropping")
			if err := tx.DeleteBucket([]byte(table)); err != nil {
				return err
			}
			log.WithField("bucket", table).Debug("creating")
			if _, err := tx.CreateBucket([]byte(table)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (client *BoltDBClient) PackageSave(pkgs ...*domain.Package) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TablePackages))
		for _, pkg := range pkgs {
			var (
				k   = []byte(pkg.Path)
				v   = b.Get(k)
				err error
			)
			if v == nil || pkg.ID == 0 {
				id, err := b.NextSequence()
				if err != nil {
					return fmt.Errorf("getting next ID for new package %q: %s", pkg.Path, err)
				}

				pkg.ID = id
			}

			if v, err = proto.Marshal(pkg); err != nil {
				return fmt.Errorf("marshalling Package %q: %s", pkg.Path, err)
			}

			if err = b.Put(k, v); err != nil {
				return fmt.Errorf("saving Package %q: %s", pkg.Path, err)
			}
		}

		return nil
	})
}

func (client *BoltDBClient) ToCrawlAdd(entries ...*domain.ToCrawlEntry) (int, error) {
	var numNew int

	if err := client.db.Update(func(tx *bolt.Tx) error {
		for _, entry := range entries {
			k := []byte(entry.PackagePath)

			{
				b := tx.Bucket([]byte(TablePackages))

				if alreadyIndexed := b.Get(k); alreadyIndexed != nil {
					log.WithField("pkg-name", entry.PackagePath).Debug("Already indexed")
					continue
				}
			}

			b := tx.Bucket([]byte(TableToCrawl))
			if alreadyQueued := b.Get(k); alreadyQueued != nil {
				// log.WithField("pkg-name", entry.PackagePath).Debug("Already queued")
				continue
			}

			numNew++

			if entry.ID == 0 {
				id, err := b.NextSequence()
				if err != nil {
					return fmt.Errorf("getting next ID for ToCrawlEntry %q: %s", entry.PackagePath, err)
				}
				entry.ID = id
			}

			v, err := proto.Marshal(entry)
			if err != nil {
				return fmt.Errorf("marshalling ToCrawlEntry %q: %s", entry.PackagePath, err)
			}

			if err := b.Put(k, v); err != nil {
				return fmt.Errorf("saving ToCrawlEntry %q: %s", entry.PackagePath, err)
			}
		}

		return nil
	}); err != nil {
		return numNew, err
	}
	return numNew, nil
}

func (client *BoltDBClient) ToCrawlSave(entries ...*domain.ToCrawlEntry) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		for _, entry := range entries {
			var (
				b = tx.Bucket([]byte(TableToCrawl))
				k = []byte(entry.PackagePath)
			)

			if entry.ID == 0 {
				id, err := b.NextSequence()
				if err != nil {
					return fmt.Errorf("getting next ID for ToCrawlEntry %q: %s", entry.PackagePath, err)
				}
				entry.ID = id
			}

			v, err := proto.Marshal(entry)
			if err != nil {
				return fmt.Errorf("marshalling ToCrawlEntry %q: %s", entry.PackagePath, err)
			}

			if err := b.Put(k, v); err != nil {
				return fmt.Errorf("saving ToCrawlEntry %q: %s", entry.PackagePath, err)
			}
		}

		return nil
	})
}

// PackageDelete N.B. no existence check is performed.
func (client *BoltDBClient) PackageDelete(pkgPaths ...string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TablePackages))
		for _, pkgPath := range pkgPaths {
			k := []byte(pkgPath)
			if err := b.Delete(k); err != nil {
				return fmt.Errorf("deleting package %q: %s", pkgPath, err)
			}
		}
		return nil
	})
}

// CrawlQueueDelete N.B. no existence check is performed.
func (client *BoltDBClient) ToCrawlDelete(pkgPaths ...string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TableToCrawl))
		for _, pkgPath := range pkgPaths {
			k := []byte(pkgPath)
			if err := b.Delete(k); err != nil {
				return fmt.Errorf("deleting to-crawl entry %q: %s", pkgPath, err)
			}
		}
		return nil
	})
}

func (client *BoltDBClient) Package(pkgPath string) (*domain.Package, error) {
	var pkg domain.Package

	if err := client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TablePackages))
			k = []byte(pkgPath)
			v = b.Get(k)
		)

		if v == nil {
			return ErrKeyNotFound
		}

		return proto.Unmarshal(v, &pkg)
	}); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (client *BoltDBClient) Packages(fn func(pkg *domain.Package)) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TablePackages))
			c = b.Cursor()
		)

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var pkg domain.Package
			if err := proto.Unmarshal(v, &pkg); err != nil {
				return err
			}
			fn(&pkg)
		}
		return nil
	})
}

func (client *BoltDBClient) PackagesWithBreak(fn func(pkg *domain.Package) bool) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TablePackages))
			c = b.Cursor()
		)

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var pkg domain.Package
			if err := proto.Unmarshal(v, &pkg); err != nil {
				return err
			}
			if cont := fn(&pkg); !cont {
				break
			}
		}
		return nil
	})
}

func (client *BoltDBClient) ToCrawl(pkgPath string) (*domain.ToCrawlEntry, error) {
	var entry domain.ToCrawlEntry

	if err := client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TableToCrawl))
			k = []byte(pkgPath)
			v = b.Get(k)
		)

		if v == nil {
			return ErrKeyNotFound
		}

		return proto.Unmarshal(v, &entry)
	}); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (client *BoltDBClient) ToCrawls(fn func(entry *domain.ToCrawlEntry)) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TableToCrawl))
			c = b.Cursor()
		)

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry domain.ToCrawlEntry
			if err := proto.Unmarshal(v, &entry); err != nil {
				return err
			}
			fn(&entry)
		}
		return nil
	})
}

func (client *BoltDBClient) ToCrawlsWithBreak(fn func(entry *domain.ToCrawlEntry) bool) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TableToCrawl))
			c = b.Cursor()
		)

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry domain.ToCrawlEntry
			if err := proto.Unmarshal(v, &entry); err != nil {
				return err
			}
			if cont := fn(&entry); !cont {
				break
			}
		}
		return nil
	})
}

func (client *BoltDBClient) PackagesLen() (int, error) {
	var n int

	if err := client.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TablePackages))

		n = b.Stats().KeyN

		return nil
	}); err != nil {
		return 0, err
	}
	return n, nil
}

func (client *BoltDBClient) ToCrawlsLen() (int, error) {
	var n int

	if err := client.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TableToCrawl))

		n = b.Stats().KeyN

		return nil
	}); err != nil {
		return 0, err
	}
	return n, nil
}

func (client *BoltDBClient) MetaSave(key string, src interface{}) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TableMetadata))

		switch src.(type) {
		case []byte:
			return b.Put([]byte(key), src.([]byte))

		case string:
			return b.Put([]byte(key), []byte(src.(string)))

		case proto.Message:
			v, err := proto.Marshal(src.(proto.Message))
			if err != nil {
				return fmt.Errorf("marshalling %T: %s", src, err)
			}
			return b.Put([]byte(key), v)

		default:
			return ErrMetadataUnsupportedSrcType
		}
	})
}

func (client *BoltDBClient) MetaDelete(key string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TableMetadata))
		return b.Delete([]byte(key))
	})
}

func (client *BoltDBClient) Meta(key string, dst interface{}) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(TableMetadata))
			v = b.Get([]byte(key))
		)

		switch dst.(type) {
		case *[]byte:
			ptr := dst.(*[]byte)
			*ptr = v

		case *string:
			ptr := dst.(*string)
			*ptr = string(v)

		case proto.Message:
			return proto.Unmarshal(v, dst.(proto.Message))

		default:
			return ErrMetadataUnsupportedDstType
		}

		return nil
	})
}

func (client *BoltDBClient) Search(q string) (*domain.Package, error) {
	return nil, fmt.Errorf("not yet implemented")
}
