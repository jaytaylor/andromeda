package db

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"gigawatt.io/errorlib"
	"gigawatt.io/oslib"
	"github.com/coreos/bbolt"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"jaytaylor.com/bboltqueue"
	// "github.com/ulikunitz/xz"

	"jaytaylor.com/andromeda/domain"
)

var (
	ErrMetadataUnsupportedSrcType = errors.New("unsupported src type: must be an []byte, string, or proto.Message")
	ErrMetadataUnsupportedDstType = errors.New("unsupported dst type: must be an *[]byte, *string, or proto.Message")
)

type BoltConfig struct {
	DBFile      string
	BoltOptions *bolt.Options
}

func NewBoltConfig(dbFilename string) *BoltConfig {
	cfg := &BoltConfig{
		DBFile: dbFilename,
		BoltOptions: &bolt.Options{
			Timeout: 1 * time.Second,
		},
	}
	return cfg
}

func (cfg BoltConfig) Type() Type {
	return Bolt
}

type BoltClient struct {
	config *BoltConfig
	db     *bolt.DB
	q      *boltqueue.PQueue
	mu     sync.Mutex
}

func newBoltDBClient(config *BoltConfig) *BoltClient {
	client := &BoltClient{
		config: config,
	}
	return client
}

func (client *BoltClient) Open() error {
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

	client.q = boltqueue.NewPQueue(client.db)

	if err := client.initDB(); err != nil {
		return err
	}

	return nil
}

func (client *BoltClient) Close() error {
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

func (client *BoltClient) initDB() error {
	return client.db.Update(func(tx *bolt.Tx) error {
		buckets := []string{
			TableMetadata,
			TablePackages,
			TableToCrawl,
			TablePendingReferences,
		}
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return fmt.Errorf("initDB: creating bucket %q: %s", name, err)
			}
		}
		return nil
	})
}

func (client *BoltClient) EachRow(table string, fn func(k []byte, v []byte)) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(table))
			c = b.Cursor()
		)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fn(k, v)
		}
		return nil
	})
}

func (client *BoltClient) EachRowWithBreak(table string, fn func(k []byte, v []byte) bool) error {
	return client.db.View(func(tx *bolt.Tx) error {
		var (
			b = tx.Bucket([]byte(table))
			c = b.Cursor()
		)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if cont := fn(k, v); !cont {
				break
			}
		}
		return nil
	})
}

func (client *BoltClient) Purge(tables ...string) error {
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

func (client *BoltClient) PackageSave(pkgs ...*domain.Package) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		// TODO: Detect when package imports have changed, find the deleted ones, and
		//       go update those packages' imported-by to mark the import as inactive.
		return client.packageSave(tx, pkgs)
	})
}

func (client *BoltClient) packageSave(tx *bolt.Tx, pkgs []*domain.Package) error {
	b := tx.Bucket([]byte(TablePackages))
	for _, pkg := range pkgs {
		var (
			k   = []byte(pkg.Path)
			v   = b.Get(k)
			err error
		)
		if err = client.mergePendingReferences(tx, pkg); err != nil {
			return err
		}

		if v == nil || pkg.ID == 0 {
			id, err := b.NextSequence()
			if err != nil {
				return fmt.Errorf("getting next ID for new package %q: %s", pkg.Path, err)
			}

			pkg.ID = id
		}

		if len(pkg.History) > 0 {
			if pc := pkg.History[len(pkg.History)-1]; len(pc.JobMessages) > 0 {
				log.WithField("pkg", pkg.Path).Errorf("Discarding JobMessages: %v", pc.JobMessages)
				pc.JobMessages = nil
			}
		}

		if v, err = proto.Marshal(pkg); err != nil {
			return fmt.Errorf("marshalling Package %q: %s", pkg.Path, err)
		}

		if err = b.Put(k, v); err != nil {
			return fmt.Errorf("saving Package %q: %s", pkg.Path, err)
		}
	}

	return nil
}

// mergePendingReferences merges pre-existing pending references.
func (client *BoltClient) mergePendingReferences(tx *bolt.Tx, pkg *domain.Package) error {
	pendingRefs, err := client.pendingReferences(tx, pkg.Path)
	if err != nil {
		return fmt.Errorf("getting pending references for package %q: %s", pkg.Path, err)
	}
	keys := []string{}
	for _, prs := range pendingRefs {
		pkg.MergePending(prs)
		keys = append(keys, prs.PackagePath)
	}
	if err = client.pendingReferencesDelete(tx, keys); err != nil {
		return fmt.Errorf("removing prending references after merge for package %q: %s", pkg.Path, err)
	}
	log.WithField("pkg", pkg.Path).Debugf("Merged %v pending references", len(pendingRefs))
	return nil
}

// PackageDelete N.B. no existence check is performed.
func (client *BoltClient) PackageDelete(pkgPaths ...string) error {
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

var pkgSepB = []byte{'/'}

func (client *BoltClient) Package(pkgPath string) (*domain.Package, error) {
	var pkg *domain.Package

	if err := client.db.View(func(tx *bolt.Tx) error {
		var err error
		if pkg, err = client.pkg(tx, pkgPath); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (client *BoltClient) pkg(tx *bolt.Tx, pkgPath string) (*domain.Package, error) {
	var (
		b   = tx.Bucket([]byte(TablePackages))
		k   = []byte(pkgPath)
		v   = b.Get(k)
		pkg = &domain.Package{}
	)

	if len(v) == 0 {
		// Fallback to hierarchical search.
		if v = client.hierarchicalKeySearch(b, k, pkgSepB); len(v) == 0 {
			return nil, ErrKeyNotFound
		}
	}

	if err := proto.Unmarshal(v, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (client *BoltClient) Packages(pkgPaths ...string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}

	if err := client.db.View(func(tx *bolt.Tx) error {
		var err error
		if pkgs, err = client.pkgs(tx, pkgPaths); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

func (client *BoltClient) pkgs(tx *bolt.Tx, pkgPaths []string) (map[string]*domain.Package, error) {
	var (
		pkgs = map[string]*domain.Package{}
		b    = tx.Bucket([]byte(TablePackages))
	)
	for _, pkgPath := range pkgPaths {
		k := []byte(pkgPath)
		v := b.Get(k)
		if len(v) == 0 {
			// Fallback to hierarchical search.
			v = client.hierarchicalKeySearch(b, k, pkgSepB)
		}
		if len(v) > 0 {
			pkg := &domain.Package{}
			if err := proto.Unmarshal(v, pkg); err != nil {
				return pkgs, err
			}
			pkgs[pkgPath] = pkg
		}
	}
	return pkgs, nil

}

// hierarchicalKeySearch searches for any keys matching the leading part of the
// input key when split by "/" characters.
func (client *BoltClient) hierarchicalKeySearch(b *bolt.Bucket, key []byte, splitOn []byte) []byte {
	if pieces := bytes.Split(key, splitOn); len(pieces) >= 2 {
		b.ForEach(func(k, v []byte) error {
			return nil
		})
		var (
			prefix = bytes.Join(pieces[0:2], splitOn)
			v      []byte
		)
		if v = b.Get(prefix); len(v) > 0 {
			return v
		}
		for _, piece := range pieces[2:] {
			prefix = append(prefix, append(splitOn, piece...)...)
			if v = b.Get(prefix); len(v) > 0 {
				return v
			}
		}
	}
	return nil
}

func (client *BoltClient) PathPrefixSearch(prefix string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}
	err := client.db.View(func(tx *bolt.Tx) error {
		var (
			b        = tx.Bucket([]byte(TablePackages))
			prefixBs = []byte(prefix)
			c        = b.Cursor()
			errs     = []error{}
		)
		for k, v := c.Seek(prefixBs); k != nil && bytes.HasPrefix(k, prefixBs); k, v = c.Next() {
			pkg := &domain.Package{}
			if err := proto.Unmarshal(v, pkg); err != nil {
				errs = append(errs, err)
				continue
			}
			pkgs[pkg.Path] = pkg
		}
		return errorlib.Merge(errs)
	})
	if err != nil {
		return pkgs, err
	}
	if len(pkgs) == 0 {
		return nil, ErrKeyNotFound
	}
	return pkgs, nil
}

func (client *BoltClient) EachPackage(fn func(pkg *domain.Package)) error {
	var protoErr error
	if err := client.EachRow(TablePackages, func(k []byte, v []byte) {
		pkg := &domain.Package{}
		if protoErr = proto.Unmarshal(v, pkg); protoErr != nil {
			log.Errorf("Unexpected proto unmarshal error: %s", protoErr)
			protoErr = fmt.Errorf("unmarshaling pkg=%v: %s", string(k), protoErr)
		}
		fn(pkg)
	}); err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (client *BoltClient) EachPackageWithBreak(fn func(pkg *domain.Package) bool) error {
	var protoErr error
	if err := client.EachRowWithBreak(TablePackages, func(k []byte, v []byte) bool {
		pkg := &domain.Package{}
		if protoErr = proto.Unmarshal(v, pkg); protoErr != nil {
			log.Errorf("Unexpected proto unmarshal error: %s", protoErr)
			protoErr = fmt.Errorf("unmarshaling pkg=%v: %s", string(k), protoErr)
			return false
		}
		return fn(pkg)
	}); err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (client *BoltClient) PackagesLen() (int, error) {
	return client.tableLen(TablePackages)
}

func (client *BoltClient) RecordImportedBy(refPkg *domain.Package, resources map[string]*domain.PackageReferences) error {
	log.Infof("Recording imported  by... %v", len(resources))
	var (
		entries     = []*domain.ToCrawlEntry{}
		discoveries = []string{}
	)

	if err := client.db.Update(func(tx *bolt.Tx) error {
		pkgPaths := []string{}
		for pkgPath, _ := range resources {
			pkgPaths = append(pkgPaths, pkgPath)
		}
		pkgs, err := client.pkgs(tx, pkgPaths)
		if err != nil {
			return err
		}
		log.Infof("pkgs=+%v [ %v ]", pkgs, len(pkgs))

		for pkgPath, refs := range resources {
			pkg, ok := pkgs[pkgPath]
			if !ok {
				entry := &domain.ToCrawlEntry{
					PackagePath: pkgPath,
					Reason:      fmt.Sprintf("imported-by ref path=%v", refPkg.Path),
				}
				entries = append(entries, entry)
				discoveries = append(discoveries, pkgPath)
				//pkg = domain.
				continue
			}
			if pkg.ImportedBy == nil {
				pkg.ImportedBy = map[string]*domain.PackageReferences{}
			}
			for _, ref := range refs.Refs {
				if pkgRefs, ok := pkg.ImportedBy[ref.Path]; ok {
					var found bool
					for _, pkgRef := range pkgRefs.Refs {
						if pkgRef.Path == ref.Path {
							pkgRef.MarkSeen()
							found = true
							break
						}
					}
					if !found {
						pkgRefs.Refs = append(pkgRefs.Refs, ref)
					}
				} else {
					pkg.ImportedBy[ref.Path] = &domain.PackageReferences{
						Refs: []*domain.PackageReference{ref},
					}
				}
			}
		}
		pkgsSlice := make([]*domain.Package, 0, len(pkgs))
		for _, pkg := range pkgs {
			pkgsSlice = append(pkgsSlice, pkg)
		}
		return client.packageSave(tx, pkgsSlice)
	}); err != nil {
		return err
	}
	// TODO: Put all in a single transaction.
	if len(entries) > 0 {
		log.WithField("ref-pkg", refPkg.Path).WithField("to-crawls", len(entries)).Debugf("Adding discovered to-crawls: %v", discoveries)
		if _, err := client.ToCrawlAdd(entries, nil); err != nil {
			return err
		}
	}
	return nil
}

func (client *BoltClient) ToCrawlAdd(entries []*domain.ToCrawlEntry, opts *QueueOptions) (int, error) {
	if opts == nil {
		opts = NewQueueOptions()
	}

	candidates := map[string]*domain.ToCrawlEntry{}
	for _, entry := range entries {
		candidates[entry.PackagePath] = entry
	}

	var deserErr error

	if err := client.q.Scan(TableToCrawl, func(m *boltqueue.Message) {
		entry := &domain.ToCrawlEntry{}
		if deserErr != nil {
			return
		}
		if deserErr = proto.Unmarshal(m.Value, entry); deserErr != nil {
			deserErr = fmt.Errorf("unmarshalling boltqueue message: %s", deserErr)
			return
		}
		if _, ok := candidates[entry.PackagePath]; ok {
			// Already in queue.
			delete(candidates, entry.PackagePath)
		}
	}); err != nil {
		return 0, err
	}
	if deserErr != nil {
		return 0, deserErr
	}

	numNew := len(candidates)

	toAdd := make([]*boltqueue.Message, 0, numNew)
	for _, entry := range candidates {
		v, err := proto.Marshal(entry)
		if err != nil {
			return 0, err
		}
		toAdd = append(toAdd, boltqueue.NewMessageB(v))
	}

	if err := client.q.Enqueue(TableToCrawl, opts.Priority, toAdd...); err != nil {
		return 0, err
	}

	return numNew, nil
}

func (client *BoltClient) ToCrawlRemove(pkgs []string) (int, error) {
	return 0, ErrNotImplemented
}

// ToCrawlDequeue pops the next *domain.ToCrawlEntry off the from of the crawl queue.
func (client *BoltClient) ToCrawlDequeue() (*domain.ToCrawlEntry, error) {
	m, err := client.q.Dequeue(TableToCrawl)
	if err != nil {
		return nil, err
	}
	entry := &domain.ToCrawlEntry{}
	if err := proto.Unmarshal(m.Value, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (client *BoltClient) EachToCrawl(fn func(entry *domain.ToCrawlEntry)) error {
	var protoErr error
	err := client.q.Scan(TableToCrawl, func(m *boltqueue.Message) {
		if protoErr != nil {
			return
		}
		entry := &domain.ToCrawlEntry{}
		if protoErr = proto.Unmarshal(m.Value, entry); protoErr != nil {
			return
		}
		fn(entry)
	})
	if err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (client *BoltClient) EachToCrawlWithBreak(fn func(entry *domain.ToCrawlEntry) bool) error {
	var (
		keepGoing = true
		protoErr  error
	)
	err := client.q.ScanWithBreak(TableToCrawl, func(m *boltqueue.Message) bool {
		if !keepGoing {
			return keepGoing
		}
		entry := &domain.ToCrawlEntry{}
		if protoErr := proto.Unmarshal(m.Value, entry); protoErr != nil {
			keepGoing = false
			return keepGoing
		}
		if !fn(entry) {
			keepGoing = false
		}
		return keepGoing
	})
	if err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

const numPriorities = 10

func (client *BoltClient) ToCrawlsLen() (int, error) {
	total := 0
	for i := 0; i < numPriorities; i++ {
		n, err := client.q.Len(TableToCrawl, i)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// TODO: Add Get (slow, n due to iteration) and Update methods for ToCrawl.

func (client *BoltClient) MetaSave(key string, src interface{}) error {
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

func (client *BoltClient) MetaDelete(key string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(TableMetadata))
		return b.Delete([]byte(key))
	})
}

func (client *BoltClient) Meta(key string, dst interface{}) error {
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

func (client *BoltClient) Search(q string) (*domain.Package, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type BatchUpdateFunc func(pkg *domain.Package) (changed bool, err error)

func (client *BoltClient) applyBatchUpdate(fn BatchUpdateFunc) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		var (
			n         = 0
			b         = tx.Bucket([]byte(TablePackages))
			c         = b.Cursor()
			batchSize = 1000
			batch     = make([]*domain.Package, 0, batchSize)
			// batch     = []*domain.Package{} // make([]*domain.Package, 0, batchSize)
			i = 0
		)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if i%1000 == 0 {
				log.Debugf("i=%v", i)
			}
			i++
			pkg := &domain.Package{}
			if err := proto.Unmarshal(v, pkg); err != nil {
				return err
			}
			if changed, err := fn(pkg); err != nil {
				return err
			} else if changed {
				batch = append(batch, pkg)
				if len(batch) >= batchSize {
					log.WithField("this-batch", len(batch)).WithField("total-added", n).Debug("Saving batch")
					if err := client.packageSave(tx, batch); err != nil {
						return err
					}
					n += len(batch)
					batch = make([]*domain.Package, 0, batchSize)
					// batch = []*domain.Package{} // make([]*domain.Package, 0, batchSize)
				}
			}
		}
		if len(batch) > 0 {
			log.WithField("this-batch", len(batch)).WithField("total-added", n).Debug("Saving batch (final)")
			if err := client.PackageSave(batch...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (client *BoltClient) BackupTo(destFile string) error {
	return client.db.View(func(tx *bolt.Tx) error {
		if exists, err := oslib.PathExists(destFile); err != nil {
			return err
		} else if exists {
			return errors.New("destination DB file already exists")
		}
		file, err := os.OpenFile(destFile, os.O_CREATE|os.O_RDWR, os.FileMode(int(0600)))
		if err != nil {
			return err
		}
		defer file.Close()
		log.WithField("src-db", client.config.DBFile).WithField("backup-dest", destFile).Info("Backing up DB")
		n, err := tx.WriteTo(file)
		if err != nil {
			return err
		}
		log.WithField("new-db", destFile).WithField("bytes-written", n).Info("Backup finished successfully")
		return nil
		// newDB, err := bolt.Open(client.config.DBFile, 0600, client.config.BoltOptions)
		// if err != nil {
		// 	return err
		// }
		// return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		// 	log.WithField("bucket", string(name)).Debug("Cloning bucket")
		// 	if newTx, err = newDB.Begin(); err != nil {
		// 		return err
		// 	}
		// 	tx.WriteTo(

		// 	// newDB.Update(func(tx *bolt.Tx) error {
		// 	// 	var (
		// 	// 		c     = b.Cursor()
		// 	// 		newTx *bolt.Tx
		// 	// 		newB  *bolt.Bucket
		// 	// 		err   error
		// 	// 		n     int // Tracks number of pending items in the current transaction.
		// 	// 	)
		// 	// 	if newTx, err = newDB.Begin(); err != nil {
		// 	// 		return err
		// 	// 	}
		// 	// 	if newB, err = newTx.CreateBucketIfNotExists(name); err != nil {
		// 	// 		return err
		// 	// 	}
		// 	// 	for k, v := c.First(); k != nil; k, v = c.Next() {
		// 	// 		if n >= rebuildBatchSize {
		// 	// 			newTx.Bucket(
		// 	// 			if newTx, err = newDB.Begin(); err != nil {
		// 	// 				return err
		// 	// 			}
		// 	// 			n = 0
		// 	// 		}

		// 	// 	}
		// 	// })
		// })
	})
}

const rebuildBatchSize = 25000

func (client *BoltClient) RebuildTo(newDBFile string, kvFilters ...KeyValueFilterFunc) error {
	if exists, err := oslib.PathExists(newDBFile); err != nil {
		return err
	} else if exists {
		return errors.New("destination DB file already exists")
	}

	log.WithField("src-db", client.config.DBFile).WithField("new-db", newDBFile).Info("Rebuilding DB")
	newDB, err := bolt.Open(newDBFile, 0600, client.config.BoltOptions)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	defer newDB.Close()

	return client.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			log.WithField("bucket", string(name)).Debug("Rebuilding bucket")
			var (
				c     = b.Cursor()
				newTx *bolt.Tx
				newB  *bolt.Bucket
				err   error
				n     int // Tracks number of pending items in the current transaction.
				t     int // Tracks total number of items copied.
			)
			if newTx, err = newDB.Begin(true); err != nil {
				log.Error(err.Error())
				return err
			}
			if newB, err = newTx.CreateBucket(name); err != nil {
				log.Error(err.Error())
				return err
			}
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if len(kvFilters) > 0 {
					for _, kvF := range kvFilters {
						k, v = kvF(name, k, v)
					}
				}
				if n >= rebuildBatchSize {
					log.WithField("bucket", string(name)).WithField("items-in-tx", n).WithField("total-items-added", t).Debug("Committing batch")
					if err = newTx.Commit(); err != nil {
						log.Error(err.Error())
						return err
					}
					n = 0
					if newTx, err = newDB.Begin(true); err != nil {
						log.Error(err.Error())
						return err
					}
					if newB, err = newTx.CreateBucketIfNotExists(name); err != nil {
						log.Error(err.Error())
						return err
					}
				}
				if err = newB.Put(k, v); err != nil {
					log.Error(err.Error())
					return err
				}
				n++
				t++
			}
			if n > 0 || t == 0 {
				if t == 0 {
					log.WithField("bucket", string(name)).Debug("Bucket was empty")
				} else {
					log.WithField("bucket", string(name)).WithField("items-in-tx", n).WithField("total-items-added", t).Debug("Committing batch")
				}
				if err = newTx.Commit(); err != nil {
					log.Error(err.Error())
					return err
				}
				n = 0
			}
			return nil
		})
	})
}

func (client *BoltClient) PendingReferences(pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	refs := []*domain.PendingReferences{}
	if err := client.db.View(func(tx *bolt.Tx) error {
		var err error
		if refs, err = client.pendingReferences(tx, pkgPathPrefix); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return refs, nil
}

func (client *BoltClient) pendingReferences(tx *bolt.Tx, pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	refs := []*domain.PendingReferences{}
	var (
		b        = tx.Bucket([]byte(TablePendingReferences))
		prefixBs = []byte(pkgPathPrefix)
		c        = b.Cursor()
		errs     = []error{}
	)
	for k, v := c.Seek(prefixBs); k != nil && bytes.HasPrefix(k, prefixBs); k, v = c.Next() {
		r := &domain.PendingReferences{}
		if err := proto.Unmarshal(v, r); err != nil {
			errs = append(errs, err)
			continue
		}
		refs = append(refs, r)
	}
	if err := errorlib.Merge(errs); err != nil {
		return nil, err
	}
	return refs, nil
}

func (client *BoltClient) PendingReferencesSave(pendingRefs ...*domain.PendingReferences) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		return client.pendingReferencesSave(tx, pendingRefs)
	})
}

func (client *BoltClient) pendingReferencesSave(tx *bolt.Tx, pendingRefs []*domain.PendingReferences) error {
	b := tx.Bucket([]byte(TablePendingReferences))
	for _, prs := range pendingRefs {
		v, err := proto.Marshal(prs)
		if err != nil {
			return fmt.Errorf("marshalling PendingReferences: %s", err)
		}
		if err = b.Put([]byte(prs.PackagePath), v); err != nil {
			return fmt.Errorf("put'ing pending refs key %q: %s", prs.PackagePath, err)
		}
	}
	return nil
}

func (client *BoltClient) PendingReferencesDelete(keys ...string) error {
	return client.db.Update(func(tx *bolt.Tx) error {
		return client.pendingReferencesDelete(tx, keys)
	})
}

func (client *BoltClient) pendingReferencesDelete(tx *bolt.Tx, keys []string) error {
	var (
		b   = tx.Bucket([]byte(TablePendingReferences))
		err error
	)
	log.Debug("Deleting %v pending-references: %+v", len(keys), keys)
	for _, key := range keys {
		if err = b.Delete([]byte(key)); err != nil {
			return err
		}
	}
	return nil
}

func (client *BoltClient) PendingReferencesLen() (int, error) {
	return client.tableLen(TablePendingReferences)
}

func (client *BoltClient) tableLen(name string) (int, error) {
	var n int

	if err := client.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))

		n = b.Stats().KeyN

		return nil
	}); err != nil {
		return 0, err
	}
	return n, nil

}

/*func compress(bs []byte) ([]byte, error) {
}

func decompress(bs []byte) ([]byte, error) {
}*/
