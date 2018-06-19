package crawler

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"gigawatt.io/errorlib"
	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/pkg/unique"
	"jaytaylor.com/andromeda/twilightzone/go/cmd/go/external/cfg"
	"jaytaylor.com/andromeda/twilightzone/go/cmd/go/external/load"
)

var (
	DefaultMaxItems      = -1
	DefaultSrcPath       = filepath.Join(os.TempDir(), "src")
	DefaultDeleteAfter   = false
	DefaultIncludeStdLib = false

	ErrStopRequested  = errors.New("stop requested")
	ErrPackageInvalid = errors.New("package structure is invalid")
)

func init() {
	// Set environment variables telling git to avoid triggering interactive
	// prompts.
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	os.Setenv("GIT_SSH_COMMAND", "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o BatchMode=yes")
}

type Config struct {
	MaxItems      int    // Maximum number of items to process.
	SrcPath       string // Location to checkout code to.
	DeleteAfter   bool   // Delete package code after analysis.
	IncludeStdLib bool   // Include standard library in associations and analysis.
}

func NewConfig() *Config {
	cfg := &Config{
		MaxItems:      DefaultMaxItems,
		SrcPath:       DefaultSrcPath,
		DeleteAfter:   DefaultDeleteAfter,
		IncludeStdLib: DefaultIncludeStdLib,
	}
	return cfg
}

// TODO: Support for picking up where last run left off.

type CrawlerCoordinator struct {
	db           db.DBClient
	crawler      *Crawler
	numProcessed int
	mu           sync.RWMutex
}

type Crawler struct {
	Config     *Config
	Processors []ProcessorFunc
}

// ProcessorFunc are functions which perofrm the actual crawling and record
// insights and hydration work.
type ProcessorFunc func(ctx *crawlerContext) error

type crawlerContext struct {
	entry  *domain.ToCrawlEntry
	rr     *vcs.RepoRoot
	pkg    *domain.Package
	stopCh chan struct{}
}

func (ctx *crawlerContext) shouldStop() bool {
	select {
	case <-ctx.stopCh:
		return true
	default:
		return false
	}
}

func NewCoordinator(dbClient db.DBClient, cfg *Config) *CrawlerCoordinator {
	cc := &CrawlerCoordinator{
		db:      dbClient,
		crawler: New(cfg),
	}
	return cc
}

// New creates and returns a new crawler instance with the supplied db client
// and configuration.
func New(cfg *Config) *Crawler {
	c := &Crawler{
		Config: cfg,
	}
	c.Processors = []ProcessorFunc{
		c.Collect,
		c.Hydrate,
	}
	return c
}

func (c *CrawlerCoordinator) Attach(stream domain.RemoteCrawlerService_AttachServer) error {
	for {
		var (
			entry *domain.ToCrawlEntry
			pkg   *domain.Package
			err   error
		)

		if entry, err = c.db.ToCrawlDequeue(); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		if err = stream.Send(entry); err != nil {
			if err2 := c.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == io.EOF {
				return nil
			}
			return err
		}

		if pkg, err = stream.Recv(); err != nil {
			if err2 := c.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == ErrStopRequested {
				return nil
			}
		}

		return func() error {
			c.mu.Lock()
			defer c.mu.Unlock()

			if err = c.CatalogImporters(pkg); err != nil {
				return err
			}

			extra := ""
			if err != nil {
				extra = fmt.Sprintf("; err=%s", err)
			}
			log.WithField("pkg", entry.PackagePath).Debugf("Package crawl received%v", extra)
			if pkg == nil {
				log.WithField("pkg", entry.PackagePath).Debug("Save skipped because pkg==nil")
			} else if err = c.db.PackageSave(pkg); err != nil {
				return err
			}
			return nil
		}()
	}
}

// Run crawls from the to-crawl queue.
func (c *CrawlerCoordinator) Run(stopCh chan struct{}) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Run starting")

	var (
		i   = 0
		err error
	)

	for ; c.crawler.Config.MaxItems <= 0 || i < c.crawler.Config.MaxItems; i++ {
		var (
			entry *domain.ToCrawlEntry
			pkg   *domain.Package
		)

		if entry, err = c.db.ToCrawlDequeue(); err != nil {
			break
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		if pkg, err = c.db.Package(entry.PackagePath); err != nil && err != db.ErrKeyNotFound {
			if err2 := c.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
		} else if err == db.ErrKeyNotFound {
			pkg = &domain.Package{
				Path: entry.PackagePath,
			}
		}

		if pkg, err = c.crawler.Do(pkg, stopCh); err != nil {
			if err2 := c.requeue(entry, err); err2 != nil {
				return errorlib.Merge([]error{err, err2})
			}
			if err == ErrStopRequested {
				break
			}
		} else if err = c.CatalogImporters(pkg); err != nil {
			break
		}

		extra := ""
		if err != nil {
			extra = fmt.Sprintf("; err=%s", err)
		}
		log.WithField("pkg", entry.PackagePath).Debugf("Package crawl finished%v", extra)
		if pkg == nil {
			log.WithField("pkg", entry.PackagePath).Debug("Save skipped because pkg==nil")
		} else if err = c.db.PackageSave(pkg); err != nil {
			break
		}

		c.mu.Lock()
		c.numProcessed++
		c.mu.Unlock()

		select {
		case <-stopCh:
			log.Debug("Stop request received")
			break
		default:
		}

		c.logStats()
	}
	log.WithField("i", i).Info("Run ended")
	if err != nil {
		if err == io.EOF {
			return nil
		}
	}
	return nil
}

// Do crawls the named packages.
func (c *CrawlerCoordinator) Do(stopCh chan struct{}, pkgs ...string) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Do starting")

	var (
		i   = 0
		err error
	)

	for ; i < len(pkgs) && c.crawler.Config.MaxItems <= 0 || i < c.crawler.Config.MaxItems; i++ {
		var pkg *domain.Package
		log.WithField("entry", fmt.Sprintf("%# v", pkgs[i])).Debug("Processing")
		if pkg, err = c.db.Package(pkgs[i]); err != nil && err != db.ErrKeyNotFound {
			return err
		} else if err == db.ErrKeyNotFound {
			pkg = &domain.Package{
				Path: pkgs[i],
			}
		}
		if pkg, err = c.crawler.Do(pkg, stopCh); err != nil && err == ErrStopRequested {
			break
		}

		extra := ""
		if err != nil {
			extra = fmt.Sprintf("; err=%s", err)
		}
		log.WithField("pkg", pkgs[i]).Debugf("Package crawl finished%v", extra)
		if pkg == nil {
			log.WithField("pkg", pkgs[i]).Debug("Save skipped because pkg==nil")
		} else if err = c.db.PackageSave(pkg); err != nil {
			break
		} else if err = c.CatalogImporters(pkg); err != nil {
			break
		}

		c.mu.Lock()
		c.numProcessed++
		c.mu.Unlock()

		select {
		case <-stopCh:
			log.Debug("Stop request received")
			break
		default:
		}
		c.logStats()
	}
	log.WithField("i", i).Info("Do ended")
	if err != nil {
		if err == io.EOF {
			return nil
		}
	}
	return nil
}

func (c *CrawlerCoordinator) CatalogImporters(pkg *domain.Package) error {
	//log.WithField("pkg", pkg.Path).WithField("imports", pkg.Data.AllImports()).Info("catalog starting")
	var (
		updatedPkgs = map[string]*domain.Package{}
		discoveries = map[string]*domain.ToCrawlEntry{}
	)
	for _, imp := range pkg.Data.AllImports() {
		rr, err := importToRepoRoot(imp, log.GetLevel() == log.DebugLevel)
		if err != nil {
			log.WithField("pkg", pkg.Path).Errorf("Failed to resolve repo for import=%v: %s", imp, err)
			continue
		}
		if err := c.updateAssociations(rr, pkg, updatedPkgs, discoveries); err != nil {
			return err
		}
	}
	// Save updated packages.
	if err := c.savePackagesMap(updatedPkgs); err != nil {
		return err
	}
	// Enqueue newly discovered packages.
	if err := c.enqueueToCrawlsMap(discoveries); err != nil {
		return err
	}
	//log.Infof("done finding rr's for pkg=%v", pkg.Path)
	return nil
}

// updateAssociations updates referenced packages for the "imported_by" portion
// of the graph.
func (c *CrawlerCoordinator) updateAssociations(rr *vcs.RepoRoot, consumerPkg *domain.Package, updatedPkgs map[string]*domain.Package, discoveries map[string]*domain.ToCrawlEntry) error {
	var (
		pkg *domain.Package
		ok  bool
		err error
	)
	if pkg, ok = updatedPkgs[rr.Root]; !ok {
		if pkg, err = c.db.Package(rr.Root); err != nil && err != db.ErrKeyNotFound {
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

func (c *CrawlerCoordinator) savePackagesMap(pkgs map[string]*domain.Package) error {
	list := make([]*domain.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		list = append(list, pkg)
	}
	if err := c.db.PackageSave(list...); err != nil {
		return err
	}
	return nil
}

func (c *CrawlerCoordinator) enqueueToCrawlsMap(toCrawls map[string]*domain.ToCrawlEntry) error {
	list := make([]*domain.ToCrawlEntry, 0, len(toCrawls))
	for _, entry := range toCrawls {
		list = append(list, entry)
	}
	if n, err := c.db.ToCrawlAdd(list...); err != nil {
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

func (c *CrawlerCoordinator) requeue(entry *domain.ToCrawlEntry, cause error) error {
	log.WithField("ToCrawlEntry", entry).Errorf("Issue crawling package, attempting re-queue due to: %s", cause)
	if _, err := c.db.ToCrawlAdd(entry); err != nil {
		log.WithField("ToCrawlEntry", entry).Errorf("Re-queueing crawling entry failed: %s", err)
		return err
	}
	log.WithField("ToCrawlEntry", entry).Debug("Re-queued OK")
	return nil
}

func (c *CrawlerCoordinator) NumProcessed() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.numProcessed
}

func (c *Crawler) Do(pkg *domain.Package, stopCh chan struct{}) (*domain.Package, error) {
	ctx := &crawlerContext{
		pkg:    pkg,
		stopCh: stopCh,
	}

	if ctx.shouldStop() {
		return nil, ErrStopRequested
	}

	rr, err := importToRepoRoot(pkg.Path, log.GetLevel() == log.DebugLevel)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// If first crawl.
	if pkg.MostlyEmpty() {
		pkg = newPackage(rr, &now)
	}

	pc := &domain.PackageCrawl{
		JobStartedAt: &now,
		Data:         &domain.PackageSnapshot{},
	}

	pkg.History = append(pkg.History, pc)

	localPath := filepath.Join(c.Config.SrcPath, rr.Root)

	if c.Config.DeleteAfter {
		defer os.RemoveAll(localPath)
	}

	ctx.pkg = pkg
	ctx.rr = rr

	for _, pFn := range c.Processors {
		pCh := make(chan error)
		go func() {
			pCh <- pFn(ctx)
		}()
		select {
		case err = <-pCh:
			if err != nil {
				if !c.errorShouldInterruptExecution(err) {
					log.WithField("pkg", ctx.pkg.Path).Warnf("Ignoring non-fatal error: %s", err)
					return ctx.pkg, nil
				}
				return ctx.pkg, err
			}
		case <-ctx.stopCh:
			return nil, ErrStopRequested
		}
	}
	return pkg, nil
}

// errorShouldInterruptExecution returns true if crawler should cease execution
// due to a particular error condition.
func (_ *Crawler) errorShouldInterruptExecution(err error) bool {
	if err != nil && err != ErrPackageInvalid {
		return true
	}
	return false
}

// Collect phase fetches information so a package can be analyzed.
func (c *Crawler) Collect(ctx *crawlerContext) error {
	//log.Debug("collect starting")
	// Skip standard library packages, because they're already in $GOROOT and
	// "go get" doesn't work with them.
	if !strings.Contains(ctx.pkg.Path, ".") {
		return nil
	}
	if err := c.get(ctx.rr); err != nil {
		finishedAt := time.Now()
		ctx.pkg.LatestCrawl().AddMessage(fmt.Sprintf("go getting: %v", err.Error()))
		ctx.pkg.LatestCrawl().JobSucceeded = false
		ctx.pkg.LatestCrawl().JobFinishedAt = &finishedAt
		return err
	}
	return nil
}

// Hydrate phase consumes collected information and artifacts to create or
// update a *domain.Package struct.
//
// If pkg is nil, it is assumed that this is the first crawl.
func (c *Crawler) Hydrate(ctx *crawlerContext) error {
	//log.Debug("hydrate starting")
	err := c.interrogate(ctx.pkg, ctx.rr)

	finishedAt := time.Now()

	ctx.pkg.LatestCrawl().JobFinishedAt = &finishedAt

	if err != nil {
		ctx.pkg.LatestCrawl().AddMessage(fmt.Sprintf("interrogating: %v", err.Error()))
		ctx.pkg.LatestCrawl().JobSucceeded = false
		return err
	}

	ctx.pkg.Data = ctx.pkg.Data.Merge(ctx.pkg.LatestCrawl().Data)
	ctx.pkg.LatestCrawl().JobSucceeded = true
	return nil
}

// get emulates `go get`.
func (c *Crawler) get(rr *vcs.RepoRoot) error {
	if err := os.MkdirAll(c.Config.SrcPath, os.FileMode(int(0755))); err != nil {
		return err
	}

	dst := filepath.Join(c.Config.SrcPath, rr.Root)
	if err := rr.VCS.Create(dst, rr.Repo); err != nil {
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
		// Retry after resetting the directory.
		if err := rr.VCS.Create(dst, rr.Repo); err != nil {
			log.WithField("pkg", rr.Root).Errorf("Problem creating / go-get'ing repo: %s", err)
			return err
		}
	}
	return nil
}

func (c *Crawler) interrogate(pkg *domain.Package, rr *vcs.RepoRoot) error {
	var (
		localPath = filepath.Join(c.Config.SrcPath, rr.Root)
		pc        = pkg.LatestCrawl()
	)

	pc.Data.Repo = rr.Repo
	pc.Data.Readme = detectReadme(localPath)

	importsMap := map[string]struct{}{}
	testImportsMap := map[string]struct{}{}

	scanDir := func(dir string) error {
		log.Infof("dir=%v b=%v", dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)))
		var goPkg *load.Package
		pkgPath := strings.Replace(dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)), "", 1)
		goPkg, err := loadPackageDynamic(c.Config.SrcPath, pkgPath)
		if err != nil {
			pkg.LatestCrawl().AddMessage(fmt.Sprintf("loading %v: %s", pkgPath, err))
			return ErrPackageInvalid
		}
		for _, imp := range goPkg.Imports {
			if pieces := strings.SplitN(imp, "/vendor/", 2); len(pieces) > 1 {
				imp = pieces[1]
			}
			if c.Config.IncludeStdLib || strings.Contains(imp, ".") {
				importsMap[imp] = struct{}{}
			}
		}
		for _, imp := range goPkg.TestImports {
			if pieces := strings.SplitN(imp, "/vendor/", 2); len(pieces) > 1 {
				imp = pieces[1]
			}
			if c.Config.IncludeStdLib || strings.Contains(imp, ".") {
				testImportsMap[imp] = struct{}{}
			}
		}
		return nil
	}

	if err := scanDir(localPath); err != nil {
		return err
	}

	dirs, err := subdirs(localPath)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		if err := scanDir(dir); err != nil {
			return err
		}
	}

	pc.Data.Imports = []string{}
	pc.Data.TestImports = []string{}

	for imp, _ := range importsMap {
		pc.Data.Imports = append(pc.Data.Imports, imp)
	}
	for imp, _ := range testImportsMap {
		pc.Data.TestImports = append(pc.Data.TestImports, imp)
	}

	size, err := dirSize(localPath)
	if err != nil {
		return err
	}
	pc.Data.Bytes = size

	return nil
}

func (c *CrawlerCoordinator) logStats() {
	pl, _ := c.db.PackagesLen()
	ql, _ := c.db.ToCrawlsLen()
	log.WithField("packages", pl).WithField("to-crawls", ql).Debug("Stats")
}

// loadPackageDynamic returns slices containing imports and test imports.
func loadPackageDynamic(parentDir string, pkgPath string) (*load.Package, error) {
	/*if len(cfg.Gopath) < 10 {
		log.Debugf("Adding path=%q to GOPATH", filepath.Dir(parentDir))
		cfg.Gopath = append(cfg.Gopath, filepath.Dir(parentDir)) //  GOROOTsrc = parentDir
	}*/
	//cfg.GOROOTsrc = parentDir
	//cfg.BuildContext.GOROOT = filepath.Dir(parentDir)
	cfg.BuildContext.GOPATH = filepath.Dir(parentDir)

	//cfg.Gopath = filepath.SplitList(cfg.BuildContext.GOPATH + ":" + parentDir)
	//defer func() { cfg.GOROOTsrc = GOROOTsrcBackup }()

	lps := load.Packages([]string{pkgPath})
	for _, lp := range lps {
		if lp.Error != nil {
			return lp, errors.New(lp.Error.Error())
		}
		return lp, nil
	}
	return nil, fmt.Errorf("no pkg found")
}

func subdirs(path string) ([]string, error) {
	dirs := []string{}
	err := filepath.Walk(path, func(p string, info os.FileInfo, _ error) error {
		if info != nil && info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if info != nil {
			if info.IsDir() && info.Name() == ".git" {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

func detectReadme(localPath string) string {
	candidates := []string{
		"README.md",
		"readme.md",
		"README.markdown",
		"readme.markdown",
		"README.mkd",
		"readme.mkd",
		"README",
		"readme",
	}

	for _, c := range candidates {
		readmePath := filepath.Join(localPath, c)
		info, err := os.Stat(readmePath)
		if err != nil {
			continue
		}
		if !info.IsDir() && info.Size() > 0 {
			data, err := ioutil.ReadFile(readmePath)
			if err != nil {
				continue
			}
			if len(data) > 0 {
				return string(data)
			}
		}
	}
	return ""
}

func newPackage(rr *vcs.RepoRoot, now *time.Time) *domain.Package {
	if now == nil {
		ts := time.Now()
		now = &ts
	}
	pkg := &domain.Package{
		FirstSeenAt: now,
		Path:        rr.Root,
		Name:        "", // TODO: package name(s)???
		URL:         rr.Repo,
		VCS:         rr.VCS.Name,
		Data:        nil,
		ImportedBy:  []string{},
		History:     []*domain.PackageCrawl{},
	}
	return pkg
}

func importToRepoRoot(pkgPath string, debug bool) (*vcs.RepoRoot, error) {
	var (
		rr  *vcs.RepoRoot
		err error
	)
	if strings.Contains(pkgPath, ".") {
		if rr, err = vcs.RepoRootForImportPath(pkgPath, true); err != nil { // TODO: Only set true when logging is verbose.
			return nil, err
		}
		rr.Repo = strings.Replace(rr.Repo, "https://github.com/", "git@github.com:", 1)
		log.Infof("root=%v repo=%v vcs=%v", rr.Root, rr.Repo, rr.VCS.Name)
	} else {
		rr = &vcs.RepoRoot{
			Repo: pkgPath,
			Root: pkgPath,
			VCS: &vcs.Cmd{
				Name: "local",
			},
		}
	}
	return rr, nil
}
