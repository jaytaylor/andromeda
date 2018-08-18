package db

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"gigawatt.io/errorlib"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	// "github.com/ulikunitz/xz"

	"jaytaylor.com/andromeda/domain"
)

type Client struct {
	be     Backend
	q      Queue
	opened map[string]struct{} // Track open resources.
	mu     sync.Mutex
}

func newClient(be Backend, q Queue) *Client {
	c := &Client{
		be:     be,
		q:      q,
		opened: map[string]struct{}{},
	}
	return c
}

type openclose interface {
	Open() error
	Close() error
}

func (c *Client) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, o := range map[string]openclose{"be": c.be, "q": c.q} {
		if _, ok := c.opened[name]; !ok {
			if err := o.Open(); err != nil {
				return err
			}
			c.opened[name] = struct{}{}
		}
	}

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, o := range map[string]openclose{"be": c.be, "q": c.q} {
		if _, ok := c.opened[name]; ok {
			if err := o.Close(); err != nil {
				return err
			}
			delete(c.opened, name)
		}
	}

	return nil
}

func (c *Client) EachRow(table string, fn func(key []byte, value []byte)) error {
	return c.be.EachRow(table, fn)
}

func (c *Client) EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error {
	return c.be.EachRowWithBreak(table, fn)
}

func (c *Client) Purge(tables ...string) error {
	return c.be.Drop(tables...)
}

func (c *Client) PackageSave(pkgs ...*domain.Package) error {
	return c.be.Update(func(tx Transaction) error {
		// TODO: Detect when package imports have changed, find the deleted ones, and
		//       go update those packages' imported-by to mark the import as inactive.
		return c.packageSave(tx, pkgs)
	})
}

func (c *Client) packageSave(tx Transaction, pkgs []*domain.Package) error {
	for _, pkg := range pkgs {
		var (
			k   = []byte(pkg.Path)
			v   []byte
			err error
		)
		/*v, err := tx.Get(TablePackages, k)
		if err != nil {
			return err
		}*/
		if err = c.mergePendingReferences(tx, pkg); err != nil {
			return err
		}

		// TODO: Figure out how to handle IDs.  Put burden on backends.
		/*if v == nil || pkg.ID == 0 {
			id, err := b.NextSequence()
			if err != nil {
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

		if v, err = proto.Marshal(pkg); err != nil {
			return fmt.Errorf("marshalling Package %q: %s", pkg.Path, err)
		}

		if err = tx.Put(TablePackages, k, v); err != nil {
			return fmt.Errorf("saving Package %q: %s", pkg.Path, err)
		}
	}
	return nil
}

// mergePendingReferences merges pre-existing pending references.
func (c *Client) mergePendingReferences(tx Transaction, pkg *domain.Package) error {
	pendingRefs, err := c.pendingReferences(tx, pkg.Path)
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
	if err = c.pendingReferencesDelete(tx, keys); err != nil {
		return fmt.Errorf("removing prending references after merge for package %q: %s", pkg.Path, err)
	}
	log.WithField("pkg", pkg.Path).Debugf("Merged %v pending references", len(pendingRefs))
	return nil
}

func stringsToBytes(ss []string) [][]byte {
	bs := make([][]byte, len(ss))
	for i, s := range ss {
		bs[i] = []byte(s)
	}
	return bs
}

// PackageDelete N.B. no existence check is performed.
func (c *Client) PackageDelete(pkgPaths ...string) error {
	return c.be.Delete(TablePackages, stringsToBytes(pkgPaths)...)
}

func (c *Client) Package(pkgPath string) (*domain.Package, error) {
	var pkg *domain.Package

	if err := c.be.View(func(tx Transaction) error {
		var err error
		if pkg, err = c.pkg(tx, pkgPath); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (c *Client) pkg(tx Transaction, pkgPath string) (*domain.Package, error) {
	pkg := &domain.Package{}
	k := []byte(pkgPath)

	v, err := c.be.Get(TablePackages, k)
	if err != nil && err != ErrKeyNotFound {
		return nil, err
	}

	if len(v) == 0 {
		// Fallback to hierarchical search.
		if v, err = c.hierarchicalKeySearch(tx, k, pkgSepB); err != nil {
			return nil, err
		} else if len(v) == 0 {
			return nil, ErrKeyNotFound
		}
	}

	if err := proto.Unmarshal(v, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (c *Client) Packages(pkgPaths ...string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}

	if err := c.be.View(func(tx Transaction) error {
		var err error
		if pkgs, err = c.pkgs(tx, pkgPaths); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

func (c *Client) pkgs(tx Transaction, pkgPaths []string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}

	for _, pkgPath := range pkgPaths {
		k := []byte(pkgPath)
		v, err := tx.Get(TablePackages, k)
		if err != nil && err != ErrKeyNotFound {
			return pkgs, err
		}
		if len(v) == 0 {
			// Fallback to hierarchical search.
			if v, err = c.hierarchicalKeySearch(tx, k, pkgSepB); err != nil && err != ErrKeyNotFound {
				return pkgs, err
			}
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
func (c *Client) hierarchicalKeySearch(tx Transaction, key []byte, splitOn []byte) ([]byte, error) {
	if pieces := bytes.Split(key, splitOn); len(pieces) >= 2 {
		prefix := bytes.Join(pieces[0:2], splitOn)
		v, err := tx.Get(TablePackages, prefix)
		if err != nil && err != ErrKeyNotFound {
			return nil, err
		} else if len(v) > 0 {
			return v, nil
		}
		for _, piece := range pieces[2:] {
			prefix = append(prefix, append(splitOn, piece...)...)
			if v, err = tx.Get(TablePackages, prefix); err != nil && err != ErrKeyNotFound {
				return nil, err
			} else if len(v) > 0 {
				return v, nil
			}
		}
	}
	return nil, ErrKeyNotFound
}

// PathPrefixSearch finds all packages with a given prefix.
func (c *Client) PathPrefixSearch(prefix string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}
	err := c.be.Update(func(tx Transaction) error {
		var (
			prefixBs = []byte(prefix)
			cursor   = tx.Cursor(TablePackages)
			errs     = []error{}
		)
		defer cursor.Close()
		for k, v := cursor.Seek(prefixBs).Data(); cursor.Err() == nil && k != nil && bytes.HasPrefix(k, prefixBs); k, v = cursor.Next().Data() {
			pkg := &domain.Package{}
			if err := proto.Unmarshal(v, pkg); err != nil {
				errs = append(errs, err)
				continue
			}
			pkgs[pkg.Path] = pkg
		}
		if err := errorlib.Merge(errs); err != nil {
			return err
		}
		if err := cursor.Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return pkgs, err
	}
	if len(pkgs) == 0 {
		return nil, ErrKeyNotFound
	}
	return pkgs, nil
}

func (c *Client) EachPackage(fn func(pkg *domain.Package)) error {
	var protoErr error
	if err := c.EachRow(TablePackages, func(k []byte, v []byte) {
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

func (c *Client) EachPackageWithBreak(fn func(pkg *domain.Package) bool) error {
	var protoErr error
	if err := c.EachRowWithBreak(TablePackages, func(k []byte, v []byte) bool {
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

func (c *Client) PackagesLen() (int, error) {
	return c.be.Len(TablePackages)
}

func (c *Client) RecordImportedBy(refPkg *domain.Package, resources map[string]*domain.PackageReferences) error {
	log.WithField("referenced-pkg", refPkg.Path).Infof("Recording imported by; n-resources=%v", len(resources))
	var (
		entries     = []*domain.ToCrawlEntry{}
		discoveries = []string{}
	)

	if err := c.be.Update(func(tx Transaction) error {
		pkgPaths := []string{}
		for pkgPath, _ := range resources {
			pkgPaths = append(pkgPaths, pkgPath)
		}
		pkgs, err := c.pkgs(tx, pkgPaths)
		if err != nil {
			return err
		}
		log.WithField("referenced-pkg", refPkg.Path).Infof("%v/%v importing packages already exist in the %v table", len(pkgs), len(pkgPaths), TablePackages)

		for pkgPath, refs := range resources {
			pkg, ok := pkgs[pkgPath]
			if !ok {
				// Submit for a future crawl and add a pending reference.
				entry := &domain.ToCrawlEntry{
					PackagePath: pkgPath,
					Reason:      fmt.Sprintf("imported-by ref path=%v", refPkg.Path),
				}
				entries = append(entries, entry)
				discoveries = append(discoveries, pkgPath)
				if err = c.pendingReferenceAdd(tx, refPkg, pkgPath); err != nil {
					return err
				}
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
		return c.packageSave(tx, pkgsSlice)
	}); err != nil {
		return err
	}
	// TODO: Put all in a single transaction.
	if len(entries) > 0 {
		log.WithField("ref-pkg", refPkg.Path).WithField("to-crawls", len(entries)).Debugf("Adding discovered to-crawls: %v", discoveries)
		if _, err := c.ToCrawlAdd(entries, nil); err != nil {
			return err
		}
	}
	return nil
}

// pendingReferenceAdd merges or adds a new pending reference into the pending-references table.
func (c *Client) pendingReferenceAdd(tx Transaction, refPkg *domain.Package, pkgPath string) error {
	existing, err := c.pendingReferences(tx, pkgPath)
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
	if err = c.pendingReferencesSave(tx, []*domain.PendingReferences{prs}); err != nil {
		return err
	}
	return nil
}

func (c *Client) ToCrawlAdd(entries []*domain.ToCrawlEntry, opts *QueueOptions) (int, error) {
	if opts == nil {
		opts = NewQueueOptions()
	}

	candidates := map[string]*domain.ToCrawlEntry{}
	for _, entry := range entries {
		candidates[entry.PackagePath] = entry
	}

	var deserErr error

	if err := c.q.Scan(TableToCrawl, opts, func(value []byte) {
		if deserErr != nil {
			return
		}
		entry := &domain.ToCrawlEntry{}
		if deserErr = proto.Unmarshal(value, entry); deserErr != nil {
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

	var (
		numNew = len(candidates)
		toAdd  = make([][]byte, 0, numNew)
	)

	for _, entry := range candidates {
		v, err := proto.Marshal(entry)
		if err != nil {
			return 0, err
		}
		toAdd = append(toAdd, v)
	}

	if err := c.q.Enqueue(TableToCrawl, opts.Priority, toAdd...); err != nil {
		return 0, err
	}

	return numNew, nil
}

func (c *Client) ToCrawlRemove(pkgs []string) (int, error) {
	return 0, ErrNotImplemented
}

// ToCrawlDequeue pops the next *domain.ToCrawlEntry off the from of the crawl queue.
func (c *Client) ToCrawlDequeue() (*domain.ToCrawlEntry, error) {
	value, err := c.q.Dequeue(TableToCrawl)
	if err != nil {
		return nil, err
	}
	entry := &domain.ToCrawlEntry{}
	if err := proto.Unmarshal(value, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *Client) EachToCrawl(fn func(entry *domain.ToCrawlEntry)) error {
	var protoErr error
	err := c.q.Scan(TableToCrawl, nil, func(value []byte) {
		if protoErr != nil {
			return
		}
		entry := &domain.ToCrawlEntry{}
		if protoErr = proto.Unmarshal(value, entry); protoErr != nil {
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

func (c *Client) EachToCrawlWithBreak(fn func(entry *domain.ToCrawlEntry) bool) error {
	var (
		keepGoing = true
		protoErr  error
	)
	err := c.q.Scan(TableToCrawl, nil, func(value []byte) {
		if !keepGoing {
			return
		}
		entry := &domain.ToCrawlEntry{}
		if protoErr := proto.Unmarshal(value, entry); protoErr != nil {
			keepGoing = false
			return
		}
		keepGoing = fn(entry)
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

func (c *Client) ToCrawlsLen() (int, error) {
	total := 0
	for i := 0; i < numPriorities; i++ {
		n, err := c.q.Len(TableToCrawl, i)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// TODO: Add Get (slow, n due to iteration) and Update methods for ToCrawl.

func (c *Client) MetaSave(key string, src interface{}) error {
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

	return c.be.Put(TableMetadata, []byte(key), v)
}

func (c *Client) MetaDelete(key string) error {
	return c.be.Delete(TableMetadata, []byte(key))
}

func (c *Client) Meta(key string, dst interface{}) error {
	v, err := c.be.Get(TableMetadata, []byte(key))
	if err != nil {
		return err
	}

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
}

func (c *Client) Search(q string) (*domain.Package, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (c *Client) applyBatchUpdate(fn BatchUpdateFunc) error {
	return c.be.Update(func(tx Transaction) error {
		var (
			n         = 0
			cursor    = tx.Cursor(TablePackages)
			batchSize = 1000
			batch     = make([]*domain.Package, 0, batchSize)
			// batch     = []*domain.Package{} // make([]*domain.Package, 0, batchSize)
			i = 0
		)
		defer cursor.Close()
		for k, v := cursor.First().Data(); cursor.Err() == nil && k != nil; k, v = cursor.Next().Data() {
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
					if err := c.packageSave(tx, batch); err != nil {
						return err
					}
					n += len(batch)
					batch = make([]*domain.Package, 0, batchSize)
					// batch = []*domain.Package{} // make([]*domain.Package, 0, batchSize)
				}
			}
		}
		if err := cursor.Err(); err != nil {
			return err
		}
		if len(batch) > 0 {
			log.WithField("this-batch", len(batch)).WithField("total-added", n).Debug("Saving batch (final)")
			if err := c.PackageSave(batch...); err != nil {
				return err
			}
		}
		return nil
	})
}

/*func (c *Client) BackupTo(otherClient *client) error {
	return c.be.EachTable(func(table string, tx Transaction) error {
		return otherClient.be.Update(func(otherTx Transaction) error {
			var (
				cursor = tx.Cursor(table)
				err    error
			)
			defer cursor.Close()
			for k, v := cursor.First().Data(); k != nil; k, v = cursor.Next().Data() {
				if err = otherTx.Put(table, k, v); err != nil {
					return err
				}
			}
			return nil
		})
	})
}*/

const rebuildBatchSize = 25000

func (c *Client) RebuildTo(otherClient *Client, kvFilters ...KeyValueFilterFunc) error {
	if err := c.copyBackend(otherClient, kvFilters...); err != nil {
		return err
	}
	if err := c.copyQueue(otherClient); err != nil {
		return err
	}
	return nil
}

func (c *Client) copyBackend(otherClient *Client, kvFilters ...KeyValueFilterFunc) error {
	return c.be.View(func(tx Transaction) error {
		tables := []string{
			TableMetadata,
			TablePackages,
			TablePendingReferences,
		}

		for _, table := range tables {
			otherTx, err := otherClient.be.Begin(true)
			if err != nil {
				return err
			}
			log.WithField("table", table).Debug("Rebuild starting")
			var (
				//cursor = tx.Cursor(table)
				n int // Tracks number of pending items in the current transaction.
				t int // Tracks total number of items copied.
			)
			//defer cursor.Close()
			//for k, v := cursor.First().Data(); cursor.Err() == nil && k != nil; k, v = cursor.Next().Data()
			if eachErr := c.be.EachRow(table, func(k []byte, v []byte) {
				if err != nil {
					return
				}
				// TODO : Remove pointless "if" condition.
				if len(kvFilters) > 0 {
					for _, kvF := range kvFilters {
						k, v = kvF([]byte(table), k, v)
					}
				}
				if len(k) > 0 && len(v) > 0 {
					if n >= rebuildBatchSize {
						log.WithField("table", table).WithField("items-in-tx", n).WithField("total-items-added", t).Debug("Committing batch")
						if err = otherTx.Commit(); err != nil {
							log.Error(err.Error())
							return
						}
						n = 0
						if otherTx, err = otherClient.be.Begin(true); err != nil {
							log.Error(err.Error())
							return
						}
					}
					if err = otherTx.Put(table, k, v); err != nil {
						log.Error(err.Error())
						return
					}
					n++
					t++
				}
			}); eachErr != nil {
				return eachErr
			}
			if err != nil {
				return err
			}
			//if err := cursor.Err(); err != nil {
			//	return err
			//}
			if n > 0 || t == 0 {
				if t == 0 {
					log.WithField("table", table).Debug("Table was empty")
				} else {
					log.WithField("table", table).WithField("items-in-tx", n).WithField("total-items-added", t).Debug("Committing batch")
				}
				if err = otherTx.Commit(); err != nil {
					log.Error(err.Error())
					return err
				}
				n = 0
			} else {
				// N.B.: Don't leave TX open and hanging.
				if err = otherTx.Commit(); err != nil {
					log.Error(err.Error())
					return err
				}
			}
		}
		return nil
	})
}

func (c *Client) copyQueue(otherClient *Client) error {
	var err error
	for p := 0; p < 10; p++ {
		if scanErr := c.q.Scan(TableToCrawl, &QueueOptions{Priority: p}, func(value []byte) {
			if err != nil {
				return
			}
			err = otherClient.q.Enqueue(TableToCrawl, p, value)
		}); scanErr != nil {
			return scanErr
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) PendingReferences(pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	refs := []*domain.PendingReferences{}
	if err := c.be.View(func(tx Transaction) error {
		var err error
		if refs, err = c.pendingReferences(tx, pkgPathPrefix); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return refs, nil
}

func (c *Client) pendingReferences(tx Transaction, pkgPathPrefix string) ([]*domain.PendingReferences, error) {
	refs := []*domain.PendingReferences{}
	var (
		prefixBs = []byte(pkgPathPrefix)
		cursor   = tx.Cursor(TablePendingReferences)
		errs     = []error{}
	)
	defer cursor.Close()
	for k, v := cursor.Seek(prefixBs).Data(); cursor.Err() == nil && k != nil && bytes.HasPrefix(k, prefixBs); k, v = cursor.Next().Data() {
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
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}

func (c *Client) PendingReferencesSave(pendingRefs ...*domain.PendingReferences) error {
	return c.be.Update(func(tx Transaction) error {
		return c.pendingReferencesSave(tx, pendingRefs)
	})
}

func (c *Client) pendingReferencesSave(tx Transaction, pendingRefs []*domain.PendingReferences) error {
	for _, prs := range pendingRefs {
		v, err := proto.Marshal(prs)
		if err != nil {
			return fmt.Errorf("marshalling PendingReferences: %s", err)
		}
		if err = tx.Put(TablePendingReferences, []byte(prs.PackagePath), v); err != nil {
			return fmt.Errorf("put'ing pending refs key %q: %s", prs.PackagePath, err)
		}
	}
	return nil
}

func (c *Client) PendingReferencesDelete(keys ...string) error {
	return c.be.Update(func(tx Transaction) error {
		return c.pendingReferencesDelete(tx, keys)
	})
}

func (c *Client) pendingReferencesDelete(tx Transaction, keys []string) error {
	var err error
	log.Debugf("Deleting %v pending-references: %+v", len(keys), keys)
	for _, key := range keys {
		if err = tx.Delete(TablePendingReferences, []byte(key)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) PendingReferencesLen() (int, error) {
	return c.be.Len(TablePendingReferences)
}

func (c *Client) EachPendingReferences(fn func(pendingRefs *domain.PendingReferences)) error {
	var protoErr error
	if err := c.EachRow(TablePendingReferences, func(k []byte, v []byte) {
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

func (c *Client) EachPendingReferencesWithBreak(fn func(pendingRefs *domain.PendingReferences) bool) error {
	var protoErr error
	if err := c.EachRowWithBreak(TablePendingReferences, func(k []byte, v []byte) bool {
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

/*func compress(bs []byte) ([]byte, error) {
}

func decompress(bs []byte) ([]byte, error) {
}*/
