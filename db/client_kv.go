package db

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"gigawatt.io/errorlib"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	// "github.com/ulikunitz/xz"

	"jaytaylor.com/andromeda/domain"
)

type ClientKV struct {
	be     Backend
	q      Queue
	opened map[string]struct{} // Track open resources.
	mu     sync.Mutex
}

func newClient(be Backend, q Queue) *ClientKV {
	c := &ClientKV{
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

func (c *ClientKV) Open() error {
	log.Info("ClientKV opening..")
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

	log.Info("ClientKV opened")
	return nil
}

func (c *ClientKV) Close() error {
	log.Info("ClientKV closing..")
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

	log.Info("ClientKV closed")
	return nil
}

func (c *ClientKV) EachRow(table string, fn func(key []byte, value []byte)) error {
	return c.be.EachRow(table, fn)
}

func (c *ClientKV) EachRowWithBreak(table string, fn func(key []byte, value []byte) bool) error {
	return c.be.EachRowWithBreak(table, fn)
}

func (c *ClientKV) Destroy(args ...string) error {
	var (
		kvTables = []string{}
		queues   = []string{}
	)
	for _, arg := range args {
		if IsKV(arg) {
			kvTables = append(kvTables, arg)
		} else if IsQ(arg) {
			queues = append(queues, arg)
		} else {
			return fmt.Errorf("unrecognized table or queue name %q", arg)
		}
	}
	if err := c.Backend().Destroy(kvTables...); err != nil {
		plural := ""
		if len(queues) > 0 {
			plural = "s"
		}
		return fmt.Errorf("destroying table%v: %s", plural, err)
	}
	if err := c.Queue().Destroy(queues...); err != nil {
		plural := ""
		if len(queues) > 0 {
			plural = "s"
		}
		return fmt.Errorf("destroying queue%v: %s", plural, err)
	}
	return nil
}

func (c *ClientKV) PackageSave(pkgs ...*domain.Package) error {
	return c.be.Update(func(tx Transaction) error {
		// TODO: Detect when package imports have changed, find the deleted ones, and
		//       go update those packages' imported-by to mark the import as inactive.
		return c.packageSave(tx, pkgs, true)
	})
}

func (c *ClientKV) packageSave(tx Transaction, pkgs []*domain.Package, mergePending bool) error {
	for _, pkg := range pkgs {
		if pkg == nil {
			log.Warnf("Skipping nil pkg entry")
			continue
		}
		log.WithField("pkg", pkg.Path).Debug("Save starting")

		if mergePending {
			// TODO: REMOVE ME
			// TEMPORARY: For figuring out what's eating so much memory.
			ioutil.WriteFile("/tmp/andromeda-saving", []byte(pkg.Path), os.FileMode(int(0600)))
		}

		var (
			k   = []byte(pkg.Path)
			v   []byte
			err error
		)
		/*v, err := tx.Get(TablePackages, k)
		if err != nil {
			return err
		}*/

		if mergePending {
			if err = c.mergePendingReferences(tx, pkg); err != nil {
				return err
			}
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
				// Disabled due to too much logspam.
				// log.WithField("pkg", pkg.Path).Debugf("Discarding JobMessages: %v", pc.JobMessages)
				pc.JobMessages = nil
			}
		}

		if pkg.Data != nil {
			pkg.Data.Sync()
			// Assuming single history entry, only.
			// This data will be restored during get package().
			if len(pkg.History) > 0 {
				pkg.History[0].Data = nil
			}
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
func (c *ClientKV) mergePendingReferences(tx Transaction, pkg *domain.Package) error {
	log.WithField("pkg", pkg.Path).Debug("Merging pending references starting")

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

// MergePendingReferences tries merges all outstanding pending references.
func (c *ClientKV) MergePendingReferences() error {
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
func (c *ClientKV) PackageDelete(pkgPaths ...string) error {
	return c.be.Delete(TablePackages, stringsToBytes(pkgPaths)...)
}

func (c *ClientKV) Package(pkgPath string) (*domain.Package, error) {
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

func (c *ClientKV) pkg(tx Transaction, pkgPath string) (*domain.Package, error) {
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
	// Populate the single historical record with latest data.
	if len(pkg.History) > 0 {
		pkg.History[0].Data = pkg.Data
	}
	return pkg, nil
}

func (c *ClientKV) Packages(pkgPaths ...string) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}

	if err := c.be.View(func(tx Transaction) error {
		var err error
		if pkgs, err = c.pkgs(tx, pkgPaths, false); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

// pkgs results will be pooled (shared pkg pointers) when pool=true.
//
// To reduce the number of SET operations required, we maintain a pool of
// pkgRoot->*domain.Package associations.
//
// Then when multiple pkgPaths reference the same package, the single
// ptr from the pool will be shared amongst all pkgPath keys.
func (c *ClientKV) pkgs(tx Transaction, pkgPaths []string, pool bool) (map[string]*domain.Package, error) {
	pkgs := map[string]*domain.Package{}

	for _, pkgPath := range pkgPaths {
		if pool {
			if pkg := c.findPkg(pkgs, pkgPath); pkg != nil {
				pkgs[pkgPath] = pkg
				continue
			}
		}
		k := []byte(pkgPath)
		v, err := tx.Get(TablePackages, k)
		if err != nil && err != ErrKeyNotFound {
			return pkgs, err
		}
		if len(v) == 0 {
			log.Debugf("Hierarchical searching for %v", pkgPath)
			// Fallback to hierarchical search.
			if v, err = c.hierarchicalKeySearch(tx, k, pkgSepB); err != nil && err != ErrKeyNotFound {
				return pkgs, err
			}
		}
		if len(v) > 0 {
			log.Debugf("Unmarshaling %v", pkgPath)
			pkg := &domain.Package{}
			if err := proto.Unmarshal(v, pkg); err != nil {
				return pkgs, err
			}
			pkgs[pkgPath] = pkg
		}
	}

	return pkgs, nil
}

func (_ *ClientKV) findPkg(pool map[string]*domain.Package, searchPkgPath string) *domain.Package {
	for pkgPath, pkg := range pool {
		if strings.HasPrefix(searchPkgPath, pkgPath) && pkg.Contains(searchPkgPath) {
			return pkg
		}
	}
	return nil
}

// hierarchicalKeySearch searches for any keys matching the leading part of the
// input key when split by "/" characters.
func (c *ClientKV) hierarchicalKeySearch(tx Transaction, key []byte, splitOn []byte) ([]byte, error) {
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
func (c *ClientKV) PathPrefixSearch(prefix string) (map[string]*domain.Package, error) {
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

func (c *ClientKV) EachPackage(fn func(pkg *domain.Package)) error {
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

func (c *ClientKV) EachPackageWithBreak(fn func(pkg *domain.Package) bool) error {
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

func (c *ClientKV) PackagesLen() (int, error) {
	return c.be.Len(TablePackages)
}

func (c *ClientKV) RecordImportedBy(refPkg *domain.Package, resources map[string]*domain.PackageReferences) error {
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
		pkgs, err := c.pkgs(tx, pkgPaths, true)
		if err != nil {
			return err
		}
		log.WithField("referenced-pkg", refPkg.Path).Debugf("%v/%v importing packages already exist in the %v table", len(pkgs), len(pkgPaths), TablePackages)

		var (
			// Marking all seen refs all the time creates too much fan out writes and is
			// too expensive and slow, IO-wise, and is clogging the up the system.
			// So we're doing this for now instead.
			modifiedPkgs = map[string]struct{}{}
			//structuralChanges bool
		)

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
				if _, err = c.pendingReferenceAdd(tx, refPkg, pkgPath); err != nil {
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
						modifiedPkgs[pkg.Path] = struct{}{}
						//structuralChanges = true
					}
				} else {
					pkg.ImportedBy[ref.Path] = &domain.PackageReferences{
						Refs: []*domain.PackageReference{ref},
					}
					modifiedPkgs[pkg.Path] = struct{}{}
					//structuralChanges = true
				}
			}
		}
		pkgsSlice := make([]*domain.Package, 0, len(pkgs))
		for pkgPath, _ := range modifiedPkgs {
			pkgsSlice = append(pkgsSlice, pkgs[pkgPath])
		}
		//if structuralChanges
		if len(modifiedPkgs) > 0 {
			log.WithField("referenced-pkg", refPkg.Path).Debugf("Saving %v updated packages", len(pkgsSlice))
			return c.packageSave(tx, pkgsSlice, false)
		}
		log.WithField("referenced-pkg", refPkg.Path).Debugf("No structural updates; save skipped")
		return nil
	}); err != nil {
		return err
	}
	// TODO: Put all in a single transaction.
	if len(entries) > 0 {
		log.WithField("ref-pkg", refPkg.Path).WithField("to-crawls", len(entries)).Debugf("Adding discovered to-crawls: %v", discoveries)
		priority := DefaultQueuePriority - 1
		if priority <= 0 {
			priority = 1
		}
		if _, err := c.ToCrawlAdd(entries, &QueueOptions{Priority: priority}); err != nil {
			return err
		}
	}
	return nil
}

// pendingReferenceAdd merges or adds a new pending reference into the pending-references table.
func (c *ClientKV) pendingReferenceAdd(tx Transaction, refPkg *domain.Package, pkgPath string) (bool, error) {
	existing, err := c.pendingReferences(tx, pkgPath)
	if err != nil {
		return false, err
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
		if err = c.pendingReferencesSave(tx, []*domain.PendingReferences{prs}); err != nil {
			return false, err
		}
	}
	return !found, nil
}

func (c *ClientKV) CrawlResultAdd(cr *domain.CrawlResult, opts *QueueOptions) error {
	if opts == nil {
		opts = NewQueueOptions()
	}

	v, err := proto.Marshal(cr)
	if err != nil {
		return err
	}

	if err := c.q.Enqueue(TableCrawlResults, opts.Priority, v); err != nil {
		return err
	}

	return nil

}

func (c *ClientKV) CrawlResultDequeue() (*domain.CrawlResult, error) {
	value, err := c.q.Dequeue(TableCrawlResults)
	if err != nil {
		return nil, err
	}
	cr := &domain.CrawlResult{}
	if err := proto.Unmarshal(value, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (c *ClientKV) EachCrawlResult(fn func(cr *domain.CrawlResult)) error {
	var protoErr error
	err := c.q.Scan(TableCrawlResults, nil, func(value []byte) {
		if protoErr != nil {
			return
		}
		cr := &domain.CrawlResult{}
		if protoErr = proto.Unmarshal(value, cr); protoErr != nil {
			return
		}
		fn(cr)
	})
	if err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (c *ClientKV) EachCrawlResultWithBreak(fn func(cr *domain.CrawlResult) bool) error {
	var (
		keepGoing = true
		protoErr  error
	)
	err := c.q.Scan(TableCrawlResults, nil, func(value []byte) {
		if !keepGoing {
			return
		}
		cr := &domain.CrawlResult{}
		if protoErr = proto.Unmarshal(value, cr); protoErr != nil {
			keepGoing = false
			return
		}
		keepGoing = fn(cr)
	})
	if err != nil {
		return err
	}
	if protoErr != nil {
		return protoErr
	}
	return nil
}

func (c *ClientKV) CrawlResultsLen() (int, error) {
	return c.q.Len(TableCrawlResults, 0)
}

func (c *ClientKV) ToCrawlAdd(entries []*domain.ToCrawlEntry, opts *QueueOptions) (int, error) {
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

func (c *ClientKV) ToCrawlRemove(pkgs []string) (int, error) {
	return 0, ErrNotImplemented
}

// ToCrawlDequeue pops the next *domain.ToCrawlEntry off the from of the crawl queue.
func (c *ClientKV) ToCrawlDequeue() (*domain.ToCrawlEntry, error) {
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

func (c *ClientKV) EachToCrawl(fn func(entry *domain.ToCrawlEntry)) error {
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

func (c *ClientKV) EachToCrawlWithBreak(fn func(entry *domain.ToCrawlEntry) bool) error {
	var (
		keepGoing = true
		protoErr  error
	)
	err := c.q.Scan(TableToCrawl, nil, func(value []byte) {
		if !keepGoing {
			return
		}
		entry := &domain.ToCrawlEntry{}
		if protoErr = proto.Unmarshal(value, entry); protoErr != nil {
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

func (c *ClientKV) ToCrawlsLen() (int, error) {
	return c.q.Len(TableToCrawl, 0)
}

// TODO: Add Get (slow, n due to iteration) and Update methods for ToCrawl.

func (c *ClientKV) MetaSave(key string, src interface{}) error {
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

func (c *ClientKV) MetaDelete(key string) error {
	return c.be.Delete(TableMetadata, []byte(key))
}

func (c *ClientKV) Meta(key string, dst interface{}) error {
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

func (c *ClientKV) Search(q string) (*domain.Package, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// RebuildBatchSize is the number of entries to include per transaction during
// rebuild.
var RebuildBatchSize = 25000

func (c *ClientKV) RebuildTo(otherClient Client, kvFilters ...KeyValueFilterFunc) error {
	var (
		skipKV bool
		skipQ  bool
	)
	nextFilters := []KeyValueFilterFunc{}
	for _, f := range kvFilters {
		switch funcName(f) {
		case "jaytaylor.com/andromeda/db.skipKVFilterFunc":
			log.Debug("SkipKVFilter activated")
			skipKV = true
		case "jaytaylor.com/andromeda/db.skipQFilterFunc":
			log.Debug("SkipQFilter activated")
			skipQ = true
		default:
			log.Debugf("Func %q not a sentinel filter", funcName(f))
			nextFilters = append(nextFilters, f)
		}
	}
	if !skipKV {
		if err := c.copyBackend(otherClient, nextFilters...); err != nil {
			return err
		}
	}
	if !skipQ {
		if err := c.copyQueue(otherClient); err != nil {
			return err
		}
	}
	return nil
}

func funcName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (c *ClientKV) copyBackend(otherClient Client, kvFilters ...KeyValueFilterFunc) error {
	return c.be.View(func(tx Transaction) error {
		for _, table := range kvTables() {
			otherTx, err := otherClient.Backend().Begin(true)
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
				for _, kvF := range kvFilters {
					k, v = kvF([]byte(table), k, v)
				}
				if len(k) > 0 && len(v) > 0 {
					if n >= RebuildBatchSize {
						log.WithField("table", table).WithField("items-in-tx", n).WithField("total-items-added", t).Debug("Committing batch")
						if err = otherTx.Commit(); err != nil {
							log.Error(err.Error())
							return
						}
						n = 0
						if otherTx, err = otherClient.Backend().Begin(true); err != nil {
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

// copyQueue note: this implementation / API does not support preserving
// priority.  New items will be inserted in priority order with
// priority=DefaultQueuePriority.
func (c *ClientKV) copyQueue(otherClient Client) error {
	var err error
	for _, table := range qTables {
		var l int
		if l, err = c.q.Len(table, 0); err != nil {
			return err
		}
		log.WithField("table", table).WithField("len", l).Debug("Enqueueing values")
		if scanErr := c.q.Scan(table, nil, func(value []byte) {
			if err != nil {
				return
			}
			err = otherClient.Queue().Enqueue(table, DefaultQueuePriority, value)
		}); scanErr != nil {
			return scanErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClientKV) PendingReferences(pkgPathPrefix string) ([]*domain.PendingReferences, error) {
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

func (c *ClientKV) pendingReferences(tx Transaction, pkgPathPrefix string) ([]*domain.PendingReferences, error) {
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

func (c *ClientKV) PendingReferencesSave(pendingRefs ...*domain.PendingReferences) error {
	return c.be.Update(func(tx Transaction) error {
		return c.pendingReferencesSave(tx, pendingRefs)
	})
}

func (c *ClientKV) pendingReferencesSave(tx Transaction, pendingRefs []*domain.PendingReferences) error {
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

func (c *ClientKV) PendingReferencesDelete(keys ...string) error {
	return c.be.Update(func(tx Transaction) error {
		return c.pendingReferencesDelete(tx, keys)
	})
}

func (c *ClientKV) pendingReferencesDelete(tx Transaction, keys []string) error {
	var err error
	log.Debugf("Deleting %v pending-references: %+v", len(keys), keys)
	for _, key := range keys {
		if err = tx.Delete(TablePendingReferences, []byte(key)); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClientKV) PendingReferencesLen() (int, error) {
	return c.be.Len(TablePendingReferences)
}

func (c *ClientKV) EachPendingReferences(fn func(pendingRefs *domain.PendingReferences)) error {
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

func (c *ClientKV) EachPendingReferencesWithBreak(fn func(pendingRefs *domain.PendingReferences) bool) error {
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

func (c *ClientKV) Backend() Backend {
	return c.be
}

func (c *ClientKV) Queue() Queue {
	return c.q
}

/*func compress(bs []byte) ([]byte, error) {
}

func decompress(bs []byte) ([]byte, error) {
}*/
