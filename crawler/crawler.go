package crawler

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/domain"
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

// New creates and returns a new crawler instance with the supplied db client
// and configuration.
func New(cfg *Config) *Crawler {
	log.Debugf("New crawler with config=%# v", cfg)
	c := &Crawler{
		Config: cfg,
	}
	c.Processors = []ProcessorFunc{
		c.Collect,
		c.Hydrate,
	}
	return c
}

// Resolve implements the PackageResolver interface.
func (c *Crawler) Resolve(pkgPath string) (*vcs.RepoRoot, error) {
	return importToRepoRoot(pkgPath)
}

func (c *Crawler) Do(pkg *domain.Package, stopCh chan struct{}) (*domain.Package, error) {
	log.WithField("pkg", pkg.Path).Debug("Starting crawl")

	ctx := &crawlerContext{
		pkg:    pkg,
		stopCh: stopCh,
	}

	if ctx.shouldStop() {
		return nil, ErrStopRequested
	}

	rr, err := importToRepoRoot(pkg.Path)
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
				if c.errorShouldInterruptExecution(err) {
					return ctx.pkg, err
				}
				log.WithField("pkg", ctx.pkg.Path).Warnf("Ignoring non-fatal error: %s", err)
				return ctx.pkg, nil
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

	// TODO: If $dst/.git already exists, try just running "git pull origin master" on it rather than re-downloading entire thing!

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
		// TODO: Determine if this should really be os.PathSeparator or "/".
		//       Needs testing on windows.

		// Calculate the package import path by removing the first matching occurrence
		// of the configured src-path + a slash string.
		pkgPath := strings.Replace(dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)), "", 1)
		log.WithField("candidate-pkg-path", pkgPath).Debugf("Scanning for go package and imports")
		var goPkg *load.Package
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

func (c *Master) logStats() {
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

// newPackage turns a *vcs.RepoRoot into a new *domain.Package.  If now is
// omitted or nil, the current time will be used.
func newPackage(rr *vcs.RepoRoot, now ...*time.Time) *domain.Package {
	if len(now) == 0 || now[0] == nil {
		ts := time.Now()
		now = []*time.Time{&ts}
	}
	pkg := &domain.Package{
		FirstSeenAt: now[0],
		Path:        rr.Root,
		Name:        "", // TODO: package name(s)???
		URL:         rr.Repo,
		VCS:         rr.VCS.Name,
		Data:        &domain.PackageSnapshot{},
		ImportedBy:  []string{},
		History:     []*domain.PackageCrawl{},
	}
	return pkg
}

// importToRepoRoot isolates and returns a corresponding *vcs.RepoRoot for the
// named package.
//
// There are a few special rules which are applied:
//     1. When the package path is part of the standard library (i.e. has no
//        dots in it), a *vcs.RepoRoot is manually constructed and returned.
//        This has been instrumental in enabling functional unit-tests without
//        having to do excessive amounts of mocking.
//
//     2. Repository URLs containing "https://github.com/" are replaced with
//        "git@github.com:".  This is because an interactive username/password
//        prompt was stalling the crawler when checkout of a non-existent
//        package (ie. 404) was attempted.
//
//        Note: I have not yet gone back to re-verify this since adding the git
//        interactive prompt disablements to the init() function at the top of this file.
func importToRepoRoot(pkgPath string) (*vcs.RepoRoot, error) {
	var (
		rr      *vcs.RepoRoot
		err     error
		verbose = log.GetLevel() == log.DebugLevel
	)
	if strings.Contains(pkgPath, ".") {
		if rr, err = vcs.RepoRootForImportPath(pkgPath, verbose); err != nil {
			return nil, err
		}
		rr.Repo = strings.Replace(rr.Repo, "https://github.com/", "git@github.com:", 1)
		//logInfof("root=%v repo=%v vcs=%v", rr.Root, rr.Repo, rr.VCS.Name)
		// log.WithField("root", rr.Root).WithField("repo", rr.Repo).WithField("vcs", rr.VCS.Name).Debug("Found rr OK")
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
