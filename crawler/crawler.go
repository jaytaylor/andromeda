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

	ErrStopRequested = errors.New("stop requested")
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

type Crawler struct {
	Config       *Config
	Processors   []ProcessorFunc
	db           db.DBClient
	mu           sync.RWMutex
	numProcessed int
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

// New creates and returns a new crawler instance with the supplied db client
// and configuration.
func New(dbClient db.DBClient, cfg *Config) *Crawler {
	c := &Crawler{
		Config: cfg,
		db:     dbClient,
	}
	c.Processors = []ProcessorFunc{
		c.Collect,
		c.Hydrate,
		c.CatalogImporters,
	}
	return c
}

// Run crawls from the to-crawl queue.
func (c *Crawler) Run(stopCh chan struct{}) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Run starting")

	var (
		i   = 0
		err error
	)

	for ; c.Config.MaxItems <= 0 || i < c.Config.MaxItems; i++ {
		var (
			entry *domain.ToCrawlEntry
			pkg   *domain.Package
		)

		if entry, err = c.db.ToCrawlDequeue(); err != nil {
			break
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		if pkg, err = c.do(entry.PackagePath, stopCh); err != nil {
			log.WithField("ToCrawlEntry", entry).Errorf("Issue crawling package, attempting re-queue due to: %s", err)
			if _, err := c.db.ToCrawlAdd(entry); err != nil {
				log.WithField("ToCrawlEntry", entry).Errorf("Re-queueing crawling entry failed: %s", err)
				break
			} else {
				log.WithField("ToCrawlEntry", entry).Debug("Re-queued OK")
			}
			if err == ErrStopRequested {
				break
			}
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
func (c *Crawler) Do(stopCh chan struct{}, pkgs ...string) error {
	if stopCh == nil {
		log.Debug("nil stopCh received, this job will not be stoppable")
		stopCh = make(chan struct{})
	}
	//log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler.Do starting")

	var (
		i   = 0
		err error
	)

	for ; i < len(pkgs) && c.Config.MaxItems <= 0 || i < c.Config.MaxItems; i++ {
		var pkg *domain.Package
		log.WithField("entry", fmt.Sprintf("%# v", pkgs[i])).Debug("Processing")
		if pkg, err = c.do(pkgs[i], stopCh); err != nil {
			log.WithField("ToCrawlEntry", pkgs[i]).Errorf("Re-queueing entry failed: %s", err)
			if err == ErrStopRequested {
				break
			}
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

func (c *Crawler) NumProcessed() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.numProcessed
}

func (c *Crawler) do(pkgPath string, stopCh chan struct{}) (*domain.Package, error) {
	ctx := &crawlerContext{
		stopCh: stopCh,
	}

	pkg, err := c.db.Package(pkgPath)
	if err != nil && err != db.ErrKeyNotFound {
		return nil, err
	}

	if ctx.shouldStop() {
		return nil, ErrStopRequested
	}

	rr, err := importToRepoRoot(pkgPath, true) // TODO: Only set true when logging is verbose.
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// If first crawl.
	if pkg == nil {
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
				return pkg, err
			}
		case <-ctx.stopCh:
			return nil, ErrStopRequested
		}
	}
	return pkg, nil
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

func (c *Crawler) CatalogImporters(ctx *crawlerContext) error {
	//log.WithField("pkg", ctx.pkg.Path).WithField("imports", ctx.pkg.Data.AllImports()).Info("catalog starting")
	var (
		updatedPkgs = map[string]*domain.Package{}
		discoveries = map[string]*domain.ToCrawlEntry{}
	)
	for _, imp := range ctx.pkg.Data.AllImports() {
		rr, err := importToRepoRoot(imp, true) // TODO: Only set true when logging is verbose.
		if err != nil {
			log.WithField("pkg", ctx.pkg.Path).Errorf("Failed to resolve repo for import=%v: %s", imp, err)
			continue
		}
		if err := c.updateAssociations(rr, ctx, updatedPkgs, discoveries); err != nil {
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
	//log.Infof("done finding rr's for pkg=%v", ctx.pkg.Path)
	return nil
}

// updateAssociations updates referenced packages for the "imported_by" portion
// of the graph.
func (c *Crawler) updateAssociations(rr *vcs.RepoRoot, ctx *crawlerContext, updatedPkgs map[string]*domain.Package, discoveries map[string]*domain.ToCrawlEntry) error {
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
				Reason:      fmt.Sprintf("In use by %v", ctx.pkg.Path),
			}
		}
	}
	newImportedBy := unique.Strings(append(pkg.ImportedBy, ctx.pkg.Path))
	if !reflect.DeepEqual(pkg.ImportedBy, newImportedBy) {
		pkg.ImportedBy = newImportedBy
		updatedPkgs[rr.Root] = pkg
		log.WithField("imported-by", pkg.Path).WithField("pkg", ctx.pkg.Path).Debug("Discovered new association")
	}
	return nil
}

func (c *Crawler) savePackagesMap(pkgs map[string]*domain.Package) error {
	list := make([]*domain.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		list = append(list, pkg)
	}
	if err := c.db.PackageSave(list...); err != nil {
		return err
	}
	return nil
}

func (c *Crawler) enqueueToCrawlsMap(toCrawls map[string]*domain.ToCrawlEntry) error {
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

	scanDir := func(dir string) {
		log.Infof("dir=%v b=%v", dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)))
		pkgPath := strings.Replace(dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)), "", 1)
		goPkg, err := loadPackageDynamic(c.Config.SrcPath, pkgPath)
		if err != nil {
			pkg.LatestCrawl().AddMessage(fmt.Sprintf("loading %v: %s", pkgPath, err))
			return
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
	}

	scanDir(localPath)

	dirs, err := subdirs(localPath)
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		scanDir(dir)
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

func (c *Crawler) logStats() {
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
