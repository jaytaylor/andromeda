package db

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gigawatt.io/errorlib"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	rocks "github.com/tecbot/gorocksdb"

	"jaytaylor.com/andromeda/domain"
)

type RocksConfig struct {
	DBFile         string
	RocksOptions   *rocks.Options
	RocksTXOptions *rocks.TransactionDBOptions
}

func NewRocksConfig(dbFilename string) *RocksConfig {
	cfg := &RocksConfig{
		DBFile:         dbFilename,
		RocksOptions:   rocks.NewDefaultOptions(),
		RocksTXOptions: rocks.NewDefaultTransactionDBOptions(),
	}
	cfg.RocksOptions.SetCreateIfMissing(true)
	cfg.RocksOptions.SetCreateIfMissingColumnFamilies(true)
	return cfg
}

func (cfg RocksConfig) Type() Type {
	return Rocks
}

type RocksClient struct {
	config *RocksConfig
	dbs    map[string]*rocks.TransactionDB
	mu     sync.Mutex
}

func newRocksDBClient(config *RocksConfig) *RocksClient {
	client := &RocksClient{
		config: config,
	}
	client.config.Type()
	return client
}

func (client *RocksClient) Open() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.dbs != nil {
		return nil
	}
	client.dbs = map[string]*rocks.TransactionDB{}

	for _, table := range []string{
		TablePackages,
		TablePendingReferences,
		TableMetadata,
	} {
		db, err := rocks.OpenTransactionDb(client.config.RocksOptions, client.config.RocksTXOptions, table)
		if err != nil {
			return fmt.Errorf("opening db %q: %s", table, err)
		}
		client.dbs[table] = db
	}

	// client.q = boltqueue.NewPQueue(client.pkgsDB)

	if err := client.initDB(); err != nil {
		return err
	}

	return nil
}

func (client *RocksClient) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.dbs == nil {
		return nil
	}

	for _, db := range client.dbs {
		db.Close()
	}

	client.dbs = nil

	return nil
}

func (client *RocksClient) initDB() error {
	return nil
	/*
		return client.pkgsDB.Update(func(tx *bolt.Tx) error {
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
	*/
}

func (client *RocksClient) EachRow(table string, fn func(k []byte, v []byte)) error {
	// iter.SeekToFirst()
	var (
		tx   = client.newTX(table)
		iter *rocks.Iterator
	)

	for iter = tx.NewIterator(rocks.NewDefaultReadOptions()); iter.Valid(); iter.Next() {
		k := iter.Key()
		v := iter.Value()
		fn(k.Data(), v.Data())
		k.Free()
		v.Free()
	}
	iter.Close()
	return nil
}

func (client *RocksClient) EachRowWithBreak(table string, fn func(k []byte, v []byte) bool) error {
	var (
		tx   = client.newTX(table)
		iter *rocks.Iterator
	)
	for iter := tx.NewIterator(rocks.NewDefaultReadOptions()); iter.Valid(); iter.Next() {
		k := iter.Key()
		v := iter.Value()
		cont := fn(k.Data(), v.Data())
		k.Free()
		v.Free()
		if !cont {
			break
		}
	}
	iter.Close()
	return nil
}

func (client *RocksClient) Purge(tables ...string) error {
	if err := client.Close(); err != nil {
		return err
	}

	errs := []error{}
	for _, table := range tables {
		if err := rocks.DestroyDb(table, client.config.RocksOptions); err != nil {
			errs = append(errs, fmt.Errorf("purging %q: %s", table, err))
		}
	}
	if err := errorlib.Merge(errs); err != nil {
		return err
	}

	if err := client.Open(); err != nil {
		return err
	}
	return nil
}

func (client *RocksClient) PackageSave(pkgs ...*domain.Package) error {
	// TODO: Detect when package imports have changed, find the deleted ones, and
	//       go update those packages' imported-by to mark the import as inactive.
	return client.packageSave(nil, pkgs)
}

// packageSave is an internal method to save one or more packages.  A nil tx
// value can be passed in to activate internal function-local tx management.
func (client *RocksClient) packageSave(tx *rocks.Transaction, pkgs []*domain.Package) error {
	var (
		txMgmt bool
		err    error
	)

	if tx == nil {
		txMgmt = true
		tx = client.newTX(TablePackages)
	}

	for _, pkg := range pkgs {
		if err = client.mergePendingReferences(nil, pkg); err != nil {
			return err
		}
	}

	for _, pkg := range pkgs {
		k := []byte(pkg.Path)
		/*v, err := tx.Get(rocks.NewDefaultReadOptions(), k)
		if err != nil {
			if txMgmt {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
				}
			}
			return err
		}
		defer func() {
			if v != nil {
				v.Free()
			}
		}()*/
		if err = client.mergePendingReferences(nil, pkg); err != nil {
			if txMgmt {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
				}
			}
			return err
		}

		/*if v == nil || len(v.Data()) == 0 || pkg.ID == 0 {
			id, err := b.NextSequence()
			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
				}
				return fmt.Errorf("getting next ID for new package %q: %s", pkg.Path, err)
			}

			pkg.ID = id
		}*/

		if len(pkg.History) > 0 {
			if pc := pkg.History[len(pkg.History)-1]; len(pc.JobMessages) > 0 {
				log.WithField("pkg", pkg.Path).Debugf("Discarding JobMessages: %v", pc.JobMessages)
				pc.JobMessages = nil
			}
		}

		if pkg.Data != nil {
			pkg.Data.Sync()
		}

		v, err := proto.Marshal(pkg)
		if err != nil {
			if txMgmt {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
				}
			}
			return fmt.Errorf("marshalling Package %q: %s", pkg.Path, err)
		}

		if err = tx.Put(k, v); err != nil {
			if txMgmt {
				if rbErr := tx.Rollback(); rbErr != nil {
					log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
				}
			}
			return fmt.Errorf("saving Package %q: %s", pkg.Path, err)
		}
	}

	if txMgmt {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// mergePendingReferences merges pre-existing pending references.  The tx
// parameter must never be nil.
func (client *RocksClient) mergePendingReferences(tx *rocks.Transaction, pkg *domain.Package) error {
	pendingRefs, err := client.pendingReferences(tx, pkg.Path)
	if err != nil {
		return fmt.Errorf("getting pending references for package %q: %s", pkg.Path, err)
	}
	if len(pendingRefs) == 0 {
		return nil
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
func (client *RocksClient) PackageDelete(pkgPaths ...string) error {
	tx := client.newTX(TablePackages)

	for _, pkgPath := range pkgPaths {
		if err := tx.Delete([]byte(pkgPath)); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.WithField("pkgPath", pkgPath).Errorf("tx rollback failed: %s (in additional to existing deletion failure: %s)", rbErr, err)
			}
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (client *RocksClient) Package(pkgPath string) (*domain.Package, error) {
	return client.pkg(nil, pkgPath)
}

func (client *RocksClient) pkg(tx *rocks.Transaction, pkgPath string) (*domain.Package, error) {
	var (
		pkg = &domain.Package{}
		v   *rocks.Slice
		err error
	)

	if tx != nil {
		v, err = tx.Get(rocks.NewDefaultReadOptions(), []byte(pkgPath))
	} else {
		v, err = client.dbs[TablePackages].Get(rocks.NewDefaultReadOptions(), []byte(pkgPath))
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if v != nil {
			v.Free()
		}
	}()

	if v == nil || len(v.Data()) == 0 {
		if v, err = client.hierarchicalKeySearch(tx, []byte(pkgPath), pkgSepB); err != nil {
			return nil, err
		} else if v == nil || len(v.Data()) == 0 {
			return nil, ErrKeyNotFound
		}
	}

	if err := proto.Unmarshal(v.Data(), pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (client *RocksClient) Packages(pkgPaths ...string) (map[string]*domain.Package, error) {
	return client.pkgs(nil, pkgPaths)
}

func (client *RocksClient) pkgs(tx *rocks.Transaction, pkgPaths []string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}
	for _, pkgPath := range pkgPaths {
		if err := func() error {
			var (
				v   *rocks.Slice
				err error
			)
			if tx != nil {
				v, err = tx.Get(rocks.NewDefaultReadOptions(), []byte(pkgPath))
			} else {
				v, err = client.dbs[TablePackages].Get(rocks.NewDefaultReadOptions(), []byte(pkgPath))
			}
			if err != nil {
				return err
			}
			defer func() {
				if v != nil {
					v.Free()
				}
			}()
			if v == nil || len(v.Data()) == 0 {
				v.Free()
				// Fallback to hierarchical search.
				if v, err = client.hierarchicalKeySearch(tx, []byte(pkgPath), pkgSepB); err != nil {
					return err
				}
			}
			if v != nil && len(v.Data()) > 0 {
				pkg := &domain.Package{}
				if err := proto.Unmarshal(v.Data(), pkg); err != nil {
					return err
				}
				pkgs[pkgPath] = pkg
			}
			return nil
		}(); err != nil {
			return pkgs, err
		}
	}
	return pkgs, nil
}

// hierarchicalKeySearch searches for any keys matching the leading part of the
// input key when split by "/" characters.
func (client *RocksClient) hierarchicalKeySearch(tx *rocks.Transaction, key []byte, splitOn []byte) (*rocks.Slice, error) {
	if pieces := bytes.Split(key, splitOn); len(pieces) >= 2 {
		var (
			prefix = bytes.Join(pieces[0:2], splitOn)
			v      *rocks.Slice
			err    error
		)

		if tx != nil {
			v, err = tx.Get(rocks.NewDefaultReadOptions(), prefix)
		} else {
			v, err = client.dbs[TablePackages].Get(rocks.NewDefaultReadOptions(), prefix)
		}
		if err != nil {
			return nil, err
		} else if v != nil && len(v.Data()) > 0 {
			return v, nil
		}

		for _, piece := range pieces[2:] {
			prefix = append(prefix, append(splitOn, piece...)...)

			if tx != nil {
				v, err = tx.Get(rocks.NewDefaultReadOptions(), prefix)
			} else {
				v, err = client.dbs[TablePackages].Get(rocks.NewDefaultReadOptions(), prefix)
			}
			if err != nil {
				return nil, err
			} else if v != nil && len(v.Data()) > 0 {
				return v, nil
			}
			if v != nil {
				v.Free()
			}
		}
	}
	return nil, nil
}

func (client *RocksClient) PathPrefixSearch(prefix string) (map[string]*domain.Package, error) {
	var (
		pkgs = map[string]*domain.Package{}
		tx   = client.newTX(TablePackages)
		iter *rocks.Iterator
		errs = []error{}
	)

	for iter = tx.NewIterator(rocks.NewDefaultReadOptions()); iter.Valid(); iter.Next() {
		pkg := &domain.Package{}
		v := iter.Value()
		if err := proto.Unmarshal(v.Data(), pkg); err != nil {
			v.Free()
			errs = append(errs, err)
			continue
		}
		v.Free()
		pkgs[pkg.Path] = pkg
	}
	iter.Close()
	if err := errorlib.Merge(errs); err != nil {
		return pkgs, err
	}
	if len(pkgs) == 0 {
		return nil, ErrKeyNotFound
	}
	return pkgs, nil
}

func (client *RocksClient) EachPackage(fn func(pkg *domain.Package)) error {
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

func (client *RocksClient) EachPackageWithBreak(fn func(pkg *domain.Package) bool) error {
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

func (client *RocksClient) PackagesLen() (int, error) {
	return client.tableLen(TablePackages)
}

func (client *RocksClient) RecordImportedBy(refPkg *domain.Package, resources map[string]*domain.PackageReferences) error {
	log.WithField("referenced-pkg", refPkg.Path).Infof("Recording imported by; n-resources=%v", len(resources))
	var (
		// TODO: Re-enable when tocrawls are a thing
		// entries     = []*domain.ToCrawlEntry{}
		// discoveries = []string{}
		tx = client.newTX(TablePackages)
	)

	pkgPaths := []string{}
	for pkgPath, _ := range resources {
		pkgPaths = append(pkgPaths, pkgPath)
	}
	pkgs, err := client.pkgs(tx, pkgPaths)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
		}
		return err
	}
	log.WithField("referenced-pkg", refPkg.Path).Infof("%v/%v importing packages already exist in the %v table", len(pkgs), len(pkgPaths), TablePackages)

	for pkgPath, refs := range resources {
		pkg, ok := pkgs[pkgPath]
		if !ok {
			// TODO: Re-enable when tocrawls are a thing.
			/*// Submit for a future crawl and add a pending reference.
			entry := &domain.ToCrawlEntry{
				PackagePath: pkgPath,
				Reason:      fmt.Sprintf("imported-by ref path=%v", refPkg.Path),
			}
			entries = append(entries, entry)
			discoveries = append(discoveries, pkgPath)
			if err = client.pendingReferenceAdd(tx, refPkg, pkgPath); err != nil {
				client.dbs[TablePendingReferences
				return err
			}*/
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
	if err := client.packageSave(tx, pkgsSlice); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Errorf("Rollback failed: %s (other existing error: %s)", rbErr, err)
		}
	}
	// TODO: Re-enable when trcrawls are a thing.
	/*// TODO: Put all in a single transaction.
	if len(entries) > 0 {
		log.WithField("ref-pkg", refPkg.Path).WithField("to-crawls", len(entries)).Debugf("Adding discovered to-crawls: %v", discoveries)
		if _, err := client.ToCrawlAdd(entries, nil); err != nil {
			return err
		}
	}*/
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// pendingReferenceAdd merges or adds a new pending reference into the
// pending-references table.
// tx must never be nil.
func (client *RocksClient) pendingReferenceAdd(tx *rocks.Transaction, refPkg *domain.Package, pkgPath string) error {
	existing, err := client.pendingReferences(tx, pkgPath)
	if err != nil {
		return err
	}
	var (
		now   = time.Now()
		prs   *domain.PendingReferences
		found bool
	)
	if len(existing) > 0 {
		for _, prs = range existing {
			if _, found = prs.ImportedBy[refPkg.Path]; found {
				break
			}
		}
		if found {
			found = false
			for _, ref := range prs.ImportedBy[refPkg.Path].Refs {
				if ref.Path == pkgPath {
					// Check if package needs to be reactivated.
					if !ref.Active {
						ref.Active = true
						ref.LastSeenAt = &now
					}
					found = true
					break
				}
			}
			if !found {
				pr := domain.NewPackageReference(pkgPath, &now)
				prs.ImportedBy[refPkg.Path].Refs = append(prs.ImportedBy[refPkg.Path].Refs, pr)
				found = true
			}
		}
	}
	if !found {
		pr := domain.NewPackageReference(pkgPath, &now)
		prs = &domain.PendingReferences{
			PackagePath: pkgPath,
			ImportedBy: map[string]*domain.PackageReferences{
				refPkg.Path: domain.NewPackageReferences(pr),
			},
		}
	}
	if err = client.pendingReferencesSave(tx, []*domain.PendingReferences{prs}); err != nil {
		return err
	}
	return nil
}

func (client *RocksClient) ToCrawlAdd(entries []*domain.ToCrawlEntry, opts *QueueOptions) (int, error) {
	return 0, errors.New("not yet implemented")
	/*
		if opts == nil {
			opts = NewQueueOptions()
		}

		candidates := map[string]*domain.ToCrawlEntry{}
		for _, entry := range entries {
			candidates[entry.PackagePath] = entry
		}

		var deserErr error

		if err := client.q.Scan(TableToCrawl, func(m *boltqueue.Message) {
			if deserErr != nil {
				return
			}
			if m.Priority() != opts.Priority {
				return
			}

			entry := &domain.ToCrawlEntry{}
			if deserErr = proto.Unmarshal(m.Value, entry); deserErr != nil {
				deserErr = fmt.Errorf("unmarshalling boltqueue message: %s", deserErr)
				return
			}
			if candidate, ok := candidates[entry.PackagePath]; ok && !candidate.Force {
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
	*/
}

func (client *RocksClient) ToCrawlRemove(pkgs []string) (int, error) {
	return 0, ErrNotImplemented
}

// ToCrawlDequeue pops the next *domain.ToCrawlEntry off the from of the crawl queue.
func (client *RocksClient) ToCrawlDequeue() (*domain.ToCrawlEntry, error) {
	return nil, errors.New("not yet implemented")
	/*
		m, err := client.q.Dequeue(TableToCrawl)
		if err != nil {
			return nil, err
		}
		entry := &domain.ToCrawlEntry{}
		if err := proto.Unmarshal(m.Value, entry); err != nil {
			return nil, err
		}
		return entry, nil
	*/
}

func (client *RocksClient) EachToCrawl(fn func(entry *domain.ToCrawlEntry)) error {
	return errors.New("not yet implemented")
	/*
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
	*/
}

func (client *RocksClient) EachToCrawlWithBreak(fn func(entry *domain.ToCrawlEntry) bool) error {
	return errors.New("not yet implemented")
	/*
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
	*/
}

// const numPriorities = 10

func (client *RocksClient) ToCrawlsLen() (int, error) {
	return 0, errors.New("not yet implemented")
	/*
		total := 0
		for i := 0; i < numPriorities; i++ {
			n, err := client.q.Len(TableToCrawl, i)
			if err != nil {
				return 0, err
			}
			total += n
		}
		return total, nil
	*/
}

// TODO: Add Get (slow, n due to iteration) and Update methods for ToCrawl.

func (client *RocksClient) MetaSave(key string, src interface{}) error {
	var v []byte

	switch src.(type) {
	case []byte:
		v = src.([]byte)

	case string:
		v = []byte(src.(string))

	case proto.Message:
		var err error
		if v, err = proto.Marshal(src.(proto.Message)); err != nil {
			return fmt.Errorf("marshalling %T: %s", src, err)
		}

	default:
		return ErrMetadataUnsupportedSrcType
	}

	if err := client.dbs[TableMetadata].Put(rocks.NewDefaultWriteOptions(), []byte(key), v); err != nil {
		return err
	}
	return nil

}

func (client *RocksClient) MetaDelete(key string) error {
	if err := client.dbs[TableMetadata].Delete(rocks.NewDefaultWriteOptions(), []byte(key)); err != nil {
		return err
	}
	return nil
}

func (client *RocksClient) Meta(key string, dst interface{}) error {
	v, err := client.dbs[TableMetadata].Get(rocks.NewDefaultReadOptions(), []byte(key))
	if err != nil {
		return err
	}
	defer v.Free()

	switch dst.(type) {
	case *[]byte:
		ptr := dst.(*[]byte)
		*ptr = v.Data()

	case *string:
		ptr := dst.(*string)
		*ptr = string(v.Data())

	case proto.Message:
		return proto.Unmarshal(v.Data(), dst.(proto.Message))

	default:
		return ErrMetadataUnsupportedDstType
	}

	return nil
}

func (client *RocksClient) Search(q string) (*domain.Package, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// type BatchUpdateFunc func(pkg *domain.Package) (changed bool, err error)

func (client *RocksClient) applyBatchUpdate(fn BatchUpdateFunc) error {
	return fmt.Errorf("not yet implemented")
	/*return client.pkgsDB.Update(func(tx *bolt.Tx) error {
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
	})*/
}

func (client *RocksClient) BackupTo(destFile string) error {
	return fmt.Errorf("not yet implemented")
	/*return client.pkgsDB.View(func(tx *bolt.Tx) error {
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
	})*/
}

// const rebuildBatchSize = 25000

func (client *RocksClient) RebuildTo(newDBFile string, kvFilters ...KeyValueFilterFunc) error {
	return fmt.Errorf("not yet implemented")
	/*if exists, err := oslib.PathExists(newDBFile); err != nil {
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

	return client.pkgsDB.View(func(tx *bolt.Tx) error {
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
	})*/
}

func (client *RocksClient) PendingReferences(pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	return client.pendingReferences(nil, pkgPathPrefix)
}

func (client *RocksClient) pendingReferences(tx *rocks.Transaction, pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	if tx == nil {
		tx = client.newTX(TablePendingReferences)
		defer tx.Rollback()
	}
	refs := []*domain.PendingReferences{}
	var (
		errs = []error{}
		iter *rocks.Iterator
	)
	for iter = tx.NewIterator(rocks.NewDefaultReadOptions()); iter.Valid(); iter.Next() {
		k := iter.Key()
		v := iter.Value()

		r := &domain.PendingReferences{}
		if err := proto.Unmarshal(v.Data(), r); err != nil {
			errs = append(errs, err)
			continue
		}
		refs = append(refs, r)

		k.Free()
		v.Free()
	}
	iter.Close()

	if err := errorlib.Merge(errs); err != nil {
		return nil, err
	}
	return refs, nil
}

func (client *RocksClient) PendingReferencesSave(pendingRefs ...*domain.PendingReferences) error {
	return client.pendingReferencesSave(nil, pendingRefs)
}

func (client *RocksClient) pendingReferencesSave(tx *rocks.Transaction, pendingRefs []*domain.PendingReferences) error {
	for _, prs := range pendingRefs {
		bs, err := proto.Marshal(prs)
		if err != nil {
			return fmt.Errorf("marshalling PendingReferences: %s", err)
		}
		if tx != nil {
			err = tx.Put([]byte(prs.PackagePath), bs)
		} else {
			err = client.dbs[TablePendingReferences].Put(rocks.NewDefaultWriteOptions(), []byte(prs.PackagePath), bs)
		}
		if err != nil {
			return fmt.Errorf("put'ing pending refs key %q: %s", prs.PackagePath, err)
		}
	}
	return nil
}

func (client *RocksClient) PendingReferencesDelete(keys ...string) error {
	return client.pendingReferencesDelete(nil, keys)
}

func (client *RocksClient) pendingReferencesDelete(tx *rocks.Transaction, keys []string) error {
	log.Debugf("Deleting %v pending-references: %+v", len(keys), keys)
	var err error
	for _, key := range keys {
		if tx != nil {
			err = tx.Delete([]byte(key))
		} else {
			err = client.dbs[TablePendingReferences].Delete(rocks.NewDefaultWriteOptions(), []byte(key))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (client *RocksClient) PendingReferencesLen() (int, error) {
	return client.tableLen(TablePendingReferences)
}

func (client *RocksClient) EachPendingReferences(fn func(pendingRefs *domain.PendingReferences)) error {
	var protoErr error
	if err := client.EachRow(TablePendingReferences, func(k []byte, v []byte) {
		pendingRefs := &domain.PendingReferences{}
		if protoErr = proto.Unmarshal(v, pendingRefs); protoErr != nil {
			log.Errorf("Unexpected proto unmarshal error: %s", protoErr)
			protoErr = fmt.Errorf("unmarshaling pendingRefs=%v: %s", string(k), protoErr)
		}
		fn(pendingRefs)
	}); err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (client *RocksClient) EachPendingReferencesWithBreak(fn func(pendingRefs *domain.PendingReferences) bool) error {
	var protoErr error
	if err := client.EachRowWithBreak(TablePendingReferences, func(k []byte, v []byte) bool {
		pendingRefs := &domain.PendingReferences{}
		if protoErr = proto.Unmarshal(v, pendingRefs); protoErr != nil {
			log.Errorf("Unexpected proto unmarshal error: %s", protoErr)
			protoErr = fmt.Errorf("unmarshaling pendingRefs=%v: %s", string(k), protoErr)
			return false
		}
		return fn(pendingRefs)
	}); err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

// tableLen is a generalized table length getter function.
func (client *RocksClient) tableLen(table string) (int, error) {
	// TODO: consider manually tracking counts in meta.
	opts := rocks.NewDefaultOptions()
	db, err := rocks.OpenDbForReadOnly(opts, table, false)
	if err != nil {
		return 0, err
	}

	prop := db.GetProperty("rocksdb.estimate-num-keys")
	n, err := strconv.Atoi(prop)
	if err != nil {
		return 0, err
	}
	return n, nil

}

func (client *RocksClient) newTX(table string) *rocks.Transaction {
	tx := client.dbs[table].TransactionBegin(rocks.NewDefaultWriteOptions(), rocks.NewDefaultTransactionOptions(), nil)
	return tx
}
