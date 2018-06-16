package crawler

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
	"jaytaylor.com/universe/pkg/unique"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/load"
)

var (
	ErrStopRequested = errors.New("stop requested")
)

type Config struct {
	MaxItems    int    // Maximum number of items to process.
	SrcPath     string // Location to checkout code to.
	DeleteAfter bool   // Delete package code after analysis.
}

func NewConfig() *Config {
	cfg := &Config{
		SrcPath:     filepath.Join(os.TempDir(), "src"),
		DeleteAfter: false,
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

func (c *Crawler) Run(stopCh chan struct{}) error {
	log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler starting")

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
			log.Info("Issue crawling package, attempting re-queue due to: %s", err)
			if _, err := c.db.ToCrawlAdd(entry); err != nil {
				log.WithField("ToCrawlEntry", entry).Errorf("Re-queueing entry failed: %s", err)
			}
			break
		}
		log.WithField("pkg", pkg).Info("Package crawl finished")
		if err = c.db.PackageSave(pkg); err != nil {
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
	}
	log.WithField("i", i).Info("Run ended")
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

	var rr *vcs.RepoRoot
	if strings.Contains(pkgPath, ".") {
		rr, err = vcs.RepoRootForImportPath(pkgPath, true) // TODO: Only set true when logging is verbose.
		if err != nil {
			return nil, err
		}
		log.Infof("root=%[1]v\nrepo=%[2]v\nvcs=%[3]T/%[3]v\n", rr.Root, rr.Repo, rr.VCS.Name)
	} else {
		rr = &vcs.RepoRoot{
			Repo: pkgPath,
			Root: pkgPath,
			VCS: &vcs.Cmd{
				Name: "local",
			},
		}
	}

	now := time.Now()

	// If first crawl.
	if pkg == nil {
		pkg = &domain.Package{
			FirstSeenAt: &now,
			Path:        rr.Root,
			Name:        "", // TODO: package name(s)???
			URL:         rr.Repo,
			VCS:         rr.VCS.Name,
			Data:        nil,
			History:     []*domain.PackageCrawl{},
			ImportedBy:  []string{},
		}
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
	log.Debug("collect starting")
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
	log.Debug("hydrate starting")
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
	log.Debug("catalog starting")
	for _, imp := range ctx.pkg.Data.AllImports() {
		rr, err := vcs.RepoRootForImportPath(imp, true) // TODO: Only set true when logging is verbose.
		if err != nil {
			log.Error("Failed to resolve repo for import=%v: %s", imp, err)
			continue
		}
		log.Info("found rr=%# v", rr)
		if err := c.saveImportedBy(rr, ctx); err != nil {
			return err
		}
	}
	log.Info("done finding rr's")
	return nil
}

func (c *Crawler) saveImportedBy(rr *vcs.RepoRoot, ctx *crawlerContext) error {
	pkg, err := c.db.Package(rr.Root)
	if err != nil && err != db.ErrKeyNotFound {
		return err
	}
	if pkg == nil {
		pkg = &domain.Package{
			Path:       rr.Root,
			Name:       "", // TODO: package name(s)???
			URL:        rr.Repo,
			VCS:        rr.VCS.Name,
			ImportedBy: []string{},
			Data:       nil,
			History:    []*domain.PackageCrawl{},
		}
		if n, err := c.db.ToCrawlAdd(&domain.ToCrawlEntry{
			PackagePath: rr.Root,
			Reason:      fmt.Sprintf("In use by %v", ctx.pkg.Path),
		}); err != nil {
			log.Warnf("Problem enqueueing pkg=%v: %s (ignored, continuing)", rr.Root, err)
		} else if n > 0 {
			log.Infof("Added newly discovered package=%v into to-crawl queue", rr.Root)
		}
	}
	pkg.ImportedBy = unique.Strings(append(pkg.ImportedBy, ctx.pkg.Path))

	return nil
}

func (c *Crawler) get(rr *vcs.RepoRoot) error {
	dst := filepath.Join(c.Config.SrcPath, rr.Root)
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(c.Config.SrcPath, os.FileMode(int(0755))); err != nil {
		return err
	}
	if err := rr.VCS.Create(dst, rr.Repo); err != nil {
		return err
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
			if strings.Contains(imp, ".") {
				importsMap[imp] = struct{}{}
			}
		}
		for _, imp := range goPkg.TestImports {
			if pieces := strings.SplitN(imp, "/vendor/", 2); len(pieces) > 1 {
				imp = pieces[1]
			}
			if strings.Contains(imp, ".") {
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

// loadPackageDynamic returns slices containing imports and test imports.
func loadPackageDynamic(parentDir string, pkgPath string) (*load.Package, error) {
	if len(cfg.Gopath) < 10 {
		cfg.Gopath = append(cfg.Gopath, filepath.Dir(parentDir)) //  GOROOTsrc = parentDir
	}
	//cfg.GOROOTsrc = parentDir
	//cfg.BuildContext.GOROOT = filepath.Dir(parentDir)

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
