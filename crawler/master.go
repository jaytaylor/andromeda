package crawler

import (
	"fmt"
	"io"
	"reflect"
	"sync"

	"gigawatt.io/errorlib"
	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/pkg/unique"
)

// TODO: Support for picking up where last run left off.

// Master runs varuous types of crawl jobs and is responsible for merging in the
// resulting data.
//
// Considered names:
//     - Coordinator
//     - Tracker
//     - Registry
//     - Vault
//     - Grapher
//     - Orchestrator
//     - MotherShip
//     - Overseer
//     - Master
type Master struct {
	db           db.Client
	crawler      *Crawler
	numProcessed int
	mu           sync.RWMutex
}

// NewMaster constructs and returns a new Master crawler instance.
func NewMaster(dbClient db.Client, cfg *Config) *Master {
	m := &Master{
		db:      dbClient,
		crawler: New(cfg),
	}
	return m
}

// Resolve implements the PackageResolver interface.
func (m *Master) Resolve(pkgPath string) (*vcs.RepoRoot, error) {
	if !strings.Contains(pkgPath, ".") {
		return importToRepoRoot(pkgPath)
	}
	c.db.Resolve(pkgPath
}

// Attach implements the gRPC interface for crawler workers to get jobs and
// stream back results which the master is responsible for merging and storing
// in the database.
func (m *Master) Attach(stream domain.RemoteCrawlerService_AttachServer) error {
	for {
		var (
			entry *domain.ToCrawlEntry
			pkg   *domain.Package
			err   error
		)

		if entry, err = m.db.ToCrawlDequeue(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		if err = stream.Send(entry); err != nil {
			if err2 := m.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == io.EOF {
				return nil
			}
			return err
		}

		if pkg, err = stream.Recv(); err != nil {
			if err2 := m.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == ErrStopRequested {
				return nil
			}
		}

		return func() error {
			m.mu.Lock()
			defer m.mu.Unlock()

			if err = m.CatalogImporters(pkg); err != nil {
				return err
			}

			extra := ""
			if err != nil {
				extra = fmt.Sprintf("; err=%s", err)
			}
			log.WithField("pkg", entry.PackagePath).Debugf("Package crawl received%v", extra)
			if pkg == nil {
				log.WithField("pkg", entry.PackagePath).Debug("Save skipped because pkg==nil")
			} else if err = m.db.PackageSave(pkg); err != nil {
				return err
			}
			return nil
		}()
	}
}

// Run runs crawls from the to-crawl queue.
func (m *Master) Run(stopCh chan struct{}) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Run starting")

	var (
		i   = 0
		err error
	)

	for ; m.crawler.Config.MaxItems <= 0 || i < m.crawler.Config.MaxItems; i++ {
		var (
			entry *domain.ToCrawlEntry
			pkg   *domain.Package
		)

		if entry, err = m.db.ToCrawlDequeue(); err != nil {
			break
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		if pkg, err = m.db.Package(entry.PackagePath); err != nil && err != db.ErrKeyNotFound {
			if err2 := m.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
		} else if err == db.ErrKeyNotFound {
			pkg = &domain.Package{
				Path: entry.PackagePath,
			}
		}

		if pkg, err = m.crawler.Do(pkg, stopCh); err != nil {
			if err2 := m.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == ErrStopRequested {
				break
			}
		} else if err = m.CatalogImporters(pkg); err != nil {
			break
		}

		extra := ""
		if err != nil {
			extra = fmt.Sprintf("; err=%s", err)
		}
		log.WithField("pkg", entry.PackagePath).Debugf("Package crawl finished%v", extra)
		if pkg == nil {
			log.WithField("pkg", entry.PackagePath).Debug("Save skipped because pkg==nil")
		} else if err = m.db.PackageSave(pkg); err != nil {
			break
		}

		m.mu.Lock()
		m.numProcessed++
		m.mu.Unlock()

		select {
		case <-stopCh:
			log.Debug("Stop request received")
			break
		default:
		}

		m.logStats()
	}
	log.WithField("i", i).Info("Run ended")
	if err != nil {
		if err == io.EOF {
			return nil
		}
	}
	return nil
}

// Do runs crawls for the named packages.
func (m *Master) Do(stopCh chan struct{}, pkgs ...string) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Do starting")

	var (
		i   = 0
		err error
	)

	for ; i < len(pkgs) && m.crawler.Config.MaxItems <= 0 || i < m.crawler.Config.MaxItems; i++ {
		var pkg *domain.Package
		log.WithField("entry", fmt.Sprintf("%# v", pkgs[i])).Debug("Processing")
		if pkg, err = m.db.Package(pkgs[i]); err != nil && err != db.ErrKeyNotFound {
			return err
		} else if err == db.ErrKeyNotFound {
			pkg = &domain.Package{
				Path: pkgs[i],
			}
		}
		if pkg, err = m.crawler.Do(pkg, stopCh); err != nil && err == ErrStopRequested {
			break
		}

		extra := ""
		if err != nil {
			extra = fmt.Sprintf("; err=%s", err)
		}
		log.WithField("pkg", pkgs[i]).Debugf("Package crawl finished%v", extra)
		if pkg == nil {
			log.WithField("pkg", pkgs[i]).Debug("Save skipped because pkg==nil")
		} else if err = m.db.PackageSave(pkg); err != nil {
			break
		} else if err = m.CatalogImporters(pkg); err != nil {
			break
		}

		m.mu.Lock()
		m.numProcessed++
		m.mu.Unlock()

		select {
		case <-stopCh:
			log.Debug("Stop request received")
			break
		default:
		}
		m.logStats()
	}
	log.WithField("i", i).Info("Do ended")
	if err != nil {
		if err == io.EOF {
			return nil
		}
	}
	return nil
}

// CatalogImporters resolves and adds the "imported_by" association between a
// package and 3rd party packages it makes use of.
func (m *Master) CatalogImporters(pkg *domain.Package) error {
	//log.WithField("pkg", pkg.Path).WithField("imports", pkg.Data.AllImports()).Info("catalog starting")
	var (
		updatedPkgs = map[string]*domain.Package{}
		discoveries = map[string]*domain.ToCrawlEntry{}
	)
	for _, pkgPath := range pkg.Data.AllImports() {
		rr, err := importToRepoRoot(pkgPath)
		if err != nil {
			log.WithField("pkg", pkg.Path).Errorf("Failed to resolve repo for import=%v: %s", pkgPath, err)
			continue
		}
		if err = m.buildAssociations(rr, pkg, updatedPkgs, discoveries); err != nil {
			return err
		}
	}
	// Save updated packages.
	if err := m.savePackagesMap(updatedPkgs); err != nil {
		return err
	}
	// Enqueue newly discovered packages.
	if err := m.enqueueToCrawlsMap(discoveries); err != nil {
		return err
	}
	//log.Infof("done finding rr's for pkg=%v", pkg.Path)
	return nil
}

// buildAssociations updates referenced packages for the "imported_by" portion
// of the graph.
func (m *Master) buildAssociations(rr *vcs.RepoRoot, consumerPkg *domain.Package, updatedPkgs map[string]*domain.Package, discoveries map[string]*domain.ToCrawlEntry) error {
	var (
		pkg *domain.Package
		ok  bool
		err error
	)
	if pkg, ok = updatedPkgs[rr.Root]; !ok {
		if pkg, err = m.db.Package(rr.Root); err != nil && err != db.ErrKeyNotFound {
			return err
		} else if pkg == nil {
			pkg = newPackage(rr, nil)
			discoveries[rr.Root] = &domain.ToCrawlEntry{
				PackagePath: rr.Root,
				Reason:      fmt.Sprintf("In use by %v", consumerPkg.Path),
			}
		}
	}
	newImportedBy := unique.Strings(append(pkg.ImportedBy, consumerPkg.Path))
	if !reflect.DeepEqual(pkg.ImportedBy, newImportedBy) {
		pkg.ImportedBy = newImportedBy
		updatedPkgs[rr.Root] = pkg
		log.WithField("imported-by", pkg.Path).WithField("pkg", consumerPkg.Path).Debug("Discovered new association")
	}
	return nil
}

func (m *Master) savePackagesMap(pkgs map[string]*domain.Package) error {
	list := make([]*domain.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		list = append(list, pkg)
	}
	if err := m.db.PackageSave(list...); err != nil {
		return err
	}
	return nil
}

func (m *Master) enqueueToCrawlsMap(toCrawls map[string]*domain.ToCrawlEntry) error {
	list := make([]*domain.ToCrawlEntry, 0, len(toCrawls))
	for _, entry := range toCrawls {
		list = append(list, entry)
	}
	if n, err := m.db.ToCrawlAdd(list...); err != nil {
		log.Warnf("Problem enqueueing %v new candidate to-crawl entries: %s", len(toCrawls), err)
		return err
	} else if n > 0 {
		plural := ""
		if n > 1 {
			plural = "s"
		}
		log.Infof("Added %v newly discovered package%v into to-crawl queue", n, plural)
	}
	return nil
}

func (m *Master) requeue(entry *domain.ToCrawlEntry, cause error) error {
	log.WithField("ToCrawlEntry", entry).Errorf("Issue crawling package, attempting re-queue due to: %s", cause)
	if _, err := m.db.ToCrawlAdd(entry); err != nil {
		log.WithField("ToCrawlEntry", entry).Errorf("Re-queueing crawling entry failed: %s", err)
		return err
	}
	log.WithField("ToCrawlEntry", entry).Debug("Re-queued OK")
	return nil
}

func (m *Master) NumProcessed() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.numProcessed
}
