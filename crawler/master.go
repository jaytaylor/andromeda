package crawler

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"gigawatt.io/errorlib"
	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/pkg/unique"
)

// TODO: Need scheme for ensuring a write hasn't occurred to the affected
//       package since the crawler pulled the item from the queue.

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
	db         db.Client
	crawler    *Crawler
	numCrawls  int
	numRemotes int
	mu         sync.RWMutex
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
		return PackagePathToRepoRoot(pkgPath)
	}
	pkg, err := m.db.Package(pkgPath)
	if err != nil {
		return nil, err
	}
	rr := pkg.RepoRoot()
	return rr, nil
}

// Attach implements the gRPC interface for crawler workers to get jobs and
// stream back results which the master is responsible for merging and storing
// in the database.
func (m *Master) Attach(stream domain.RemoteCrawlerService_AttachServer) error {
	m.mu.Lock()
	m.numRemotes++
	m.mu.Unlock()
	log.WithField("active-remotes", m.numRemotes).Debug("Attach invoked")
	for {
		var (
			entry  *domain.ToCrawlEntry
			result *domain.CrawlResult
			err    error
		)

		if entry, err = m.db.ToCrawlDequeue(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Sending to remote crawler")
		if err = stream.Send(entry); err != nil {
			if err2 := m.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == io.EOF {
				return nil
			}
			log.WithField("pkg", entry.PackagePath).Warnf("Problem sending entry to remote crawler: %s (attach returning)", err)
			return err
		}

		// TODO: Put stopCh plumbing so process can be interrupted and entries re-queued.
		// go func() {
		// 	result, err = stream.Recv()
		// }()
		//
		// select {
		// 	case <-stopCh
		// }

		if result, err = stream.Recv(); err != nil {
			if err2 := m.requeue(entry, fmt.Errorf("remote crawler: %s", err)); err2 != nil {
				// TODO: Consider placing entries in an intermediate table while in-flight,
				//       so they'll never get lost.
				return errorlib.Merge([]error{err, err2})
			}
			if err == ErrStopRequested {
				return nil
			}
			log.WithField("pkg", entry.PackagePath).Warnf("Problem receiving crawl result: %s (attach returning)", err)
			return err
		} else if remoteErr := result.Error(); remoteErr != nil {
			log.WithField("pkg", entry.PackagePath).Errorf("Received error inside crawl result: %s", remoteErr)
			if err2 := m.requeue(entry, fmt.Errorf("remote crawler: %s", remoteErr)); err2 != nil {
				// TODO: Consider placing entries in an intermediate table while in-flight,
				//       so they'll never get lost.
				return errorlib.Merge([]error{err, err2})
			}
			continue
		}

		if err := func() error {
			// Lock to guard against data clobbering.
			m.mu.Lock()
			defer m.mu.Unlock()

			log.WithField("pkg", entry.PackagePath).Debug("Starting index update process..")

			var existing *domain.Package
			if existing, err = m.db.Package(entry.PackagePath); err != nil && err != db.ErrKeyNotFound {
				return err
			} else if existing != nil {
				result.Package = existing.Merge(result.Package)
			}

			if err = m.CatalogImporters(result.Package); err != nil {
				return err
			}
			log.WithField("pkg", result.Package.Path).Debug("Index updated with crawl result")
			m.logStats()

			/*if pkg == nil {
				log.WithField("pkg", entry.PackagePath).Debug("Save skipped because pkg==nil")
			} else if err = m.db.PackageSave(pkg); err != nil {
				return err
			}*/
			return nil
		}(); err != nil {
			log.WithField("pkg", result.Package.Path).Errorf("Hard error saving: %s", err)
			return err
		}

		m.mu.Lock()
		m.numCrawls++
		m.mu.Unlock()
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
		m.numCrawls++
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
		m.numCrawls++
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
	log.WithField("pkg", pkg.Path).Infof("catalog starting: %# v", pkg)
	// log.WithField("pkg", pkg.Path).WithField("imports", pkg.Data.AllImports()).Info("catalog starting")

	var (
		updatedPkgs = map[string]*domain.Package{
			pkg.Path: pkg,
		}
		discoveries = map[string]*domain.ToCrawlEntry{}
	)
	knownPkgs, err := m.db.Packages(pkg.Data.AllImports()...)
	if err != nil {
		return err
	}
	for _, pkgPath := range pkg.Data.AllImports() {
		var rr *vcs.RepoRoot
		usedPkg, ok := updatedPkgs[pkgPath]
		if !ok {
			if usedPkg, ok = knownPkgs[pkgPath]; !ok {
				if rr, err = PackagePathToRepoRoot(pkgPath); err != nil {
					log.WithField("pkg", pkg.Path).Errorf("Failed to resolve repo for import=%v, skipping: %s", pkgPath, err)
					continue
				}
			}
		}
		if rr == nil {
			rr = usedPkg.RepoRoot()
		}
		if err = m.buildAssociations(rr, usedPkg, pkg, updatedPkgs, discoveries); err != nil {
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
func (m *Master) buildAssociations(rr *vcs.RepoRoot, usedPkg *domain.Package, consumerPkg *domain.Package, updatedPkgs map[string]*domain.Package, discoveries map[string]*domain.ToCrawlEntry) error {
	if rr.Root == consumerPkg.RepoRoot().Root {
		log.Debugf("Skipping association between consumerPkg=%v and %v", consumerPkg.Path, rr.Root)
		return nil
	}
	var (
		ok  bool
		err error
	)
	if usedPkg == nil {
		if usedPkg, ok = updatedPkgs[rr.Root]; !ok {
			if usedPkg, err = m.db.Package(rr.Root); err != nil && err != db.ErrKeyNotFound {
				return err
			} else if usedPkg == nil {
				usedPkg = newPackage(rr, nil)
				discoveries[rr.Root] = &domain.ToCrawlEntry{
					PackagePath: rr.Root,
					Reason:      fmt.Sprintf("In use by %v", consumerPkg.Path),
				}
			}
		}
	}
	newImportedBy := unique.Strings(append(usedPkg.ImportedBy, consumerPkg.Path))
	if !reflect.DeepEqual(usedPkg.ImportedBy, newImportedBy) {
		usedPkg.ImportedBy = newImportedBy
		updatedPkgs[rr.Root] = usedPkg
		log.WithField("consumer-pkg", consumerPkg.Path).WithField("imported-pkg", usedPkg.Path).Debug("Discovered one or more new associations")
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
	log.WithField("pkg", entry.PackagePath).Debugf("Attempting re-queue entry due to: %s", cause)
	if _, err := m.db.ToCrawlAdd(entry); err != nil {
		log.WithField("pkg", entry.PackagePath).Errorf("Re-queueing failed: %s (highly undesirable, graph integrity compromised)", err)
		return err
	}
	log.WithField("pkg", entry.PackagePath).Debug("Re-queued OK")
	return nil
}

func (m *Master) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"crawls":  m.numCrawls,
		"remotes": m.numRemotes,
	}
	return stats
}
