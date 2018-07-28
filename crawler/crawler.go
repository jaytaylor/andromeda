package crawler

import (
	"errors"
	"fmt"
	godoc "go/doc"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/daviddengcn/go-index"
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
	ErrNoGoFiles      = errors.New("repository does not contain any .go files")
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
		c.ImportedBy,
	}
	return c
}

type crawlerContext struct {
	entry  *domain.ToCrawlEntry
	rr     *vcs.RepoRoot
	res    *domain.CrawlResult
	pkg    *domain.Package
	stopCh chan struct{}
}

func newCrawlerContext(pkg *domain.Package, stopCh chan struct{}) *crawlerContext {
	ctx := &crawlerContext{
		res:    domain.NewCrawlResult(pkg, nil),
		pkg:    pkg,
		stopCh: stopCh,
	}
	return ctx
}

func (ctx *crawlerContext) shouldStop() bool {
	select {
	case <-ctx.stopCh:
		return true
	default:
		return false
	}
}

// Resolve implements the PackageResolver interface.
func (c *Crawler) Resolve(pkgPath string) (*vcs.RepoRoot, error) {
	return PackagePathToRepoRoot(pkgPath)
}

func (c *Crawler) Do(pkg *domain.Package, stopCh chan struct{}) (*domain.CrawlResult, error) {
	log.WithField("pkg", pkg.Path).Debug("Starting crawl")

	ctx := newCrawlerContext(pkg, stopCh)

	if ctx.shouldStop() {
		return nil, ErrStopRequested
	}

	var (
		rr   *vcs.RepoRoot
		rrCh = make(chan error)
	)
	go func() {
		var err error
		rr, err = PackagePathToRepoRoot(pkg.Path)
		rrCh <- err
	}()
	select {
	case err := <-rrCh:
		if err != nil {
			return nil, err
		}
	case <-ctx.stopCh:
		return nil, ErrStopRequested
	}
	ctx.rr = rr
	pkg.Path = rr.Root

	now := time.Now()

	// If first crawl.
	if pkg.MostlyEmpty() {
		pkg = domain.NewPackage(rr, &now)
	}
	ctx.pkg = pkg
	ctx.res.Package = pkg

	pc := domain.NewPackageCrawl()

	pkg.History = append(pkg.History, pc)

	// localPath := filepath.Join(c.Config.SrcPath, rr.Root)

	if c.Config.DeleteAfter {
		onePastSrcPath := filepath.Join(c.Config.SrcPath, strings.SplitN(rr.Root, "/", 2)[0])
		defer os.RemoveAll(onePastSrcPath)
	}

	for _, pFn := range c.Processors {
		pCh := make(chan error)
		go func() {
			pCh <- pFn(ctx)
		}()
		select {
		case err := <-pCh:
			if err != nil {
				if c.errorInvalidatesCrawl(err) {
					ctx.res.Package = nil
					return ctx.res, err
				}
				if c.errorShouldInterruptExecution(err) {
					return ctx.res, err
				}
				log.WithField("pkg", ctx.pkg.Path).Warnf("Ignoring non-fatal error: %s", err)
				//return ctx.pkg, nil
			}
		case <-ctx.stopCh:
			return nil, ErrStopRequested
		}
	}
	return ctx.res, nil
}

// errorShouldInterruptExecution returns true if crawler should cease execution
// due to a particular error condition.
func (_ *Crawler) errorShouldInterruptExecution(err error) bool {
	if err != nil && err != ErrPackageInvalid {
		return true
	}
	return false
}

// errorInvalidatesCrawl returns true when the issue is so serious that the
// crawl result needs to be discarded.
func (_ *Crawler) errorInvalidatesCrawl(err error) bool {
	if err == ErrNoGoFiles {
		return true
	}
	return false
}

/*// CatalogImporters resolves and adds the "imported_by" association between a
// package and 3rd party packages it makes use of.
func (c *Crawler) CatalogImporters(pkg *domain.Package) error {
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
func (c *Crawler) buildAssociations(rr *vcs.RepoRoot, usedPkg *domain.Package, consumerPkg *domain.Package, updatedPkgs map[string]*domain.Package, discoveries map[string]*domain.ToCrawlEntry) error {
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
				usedPkg = domain.NewPackage(rr, nil)
				discoveries[rr.Root] = &domain.ToCrawlEntry{
					PackagePath: rr.Root,
					Reason:      fmt.Sprintf("In use by %v", consumerPkg.Path),
				}
			}
		}
	}
	newImportedBy := unique.StringsSorted(append(usedPkg.ImportedBy, consumerPkg.Path))
	if !reflect.DeepEqual(usedPkg.ImportedBy, newImportedBy) {
		usedPkg.ImportedBy = newImportedBy
		updatedPkgs[rr.Root] = usedPkg
		log.WithField("consumer-pkg", consumerPkg.Path).WithField("imported-pkg", usedPkg.Path).Debug("Discovered one or more new associations")
	}
	return nil
}*/

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

// ImportedBy chases down the reverse mappings for the current package.
func (c *Crawler) ImportedBy(ctx *crawlerContext) error {
	for subPkgPath, subPkg := range ctx.res.Package.Data.SubPackages {
		subPkgPath = domain.SubPackagePathDenormalize(ctx.res.Package.Path, subPkgPath)
		for _, imp := range subPkg.CombinedImports() {
			// Dont include self-references.
			if !strings.HasPrefix(imp, ctx.pkg.Path) {
				ref := domain.NewPackageReference(subPkgPath, ctx.res.Package.LatestCrawl().JobStartedAt)
				if _, ok := ctx.res.ImportedResources[imp]; !ok {
					ctx.res.ImportedResources[imp] = &domain.PackageReferences{}
				}
				ctx.res.ImportedResources[imp].Refs = append(ctx.res.ImportedResources[imp].Refs, ref)
			}
		}
	}
	//for _, pkgPath := range ctx.pkg.Data.AllImports() {

	//}
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

	if err := analyzeFiles(pkg.Data, localPath); err != nil {
		return err
	}

	pc.Data.Repo = rr.Repo

	// importsMap := map[string]struct{}{}
	// testImportsMap := map[string]struct{}{}

	scanDir := func(dir string) error {
		// TODO: Determine if this should really be os.PathSeparator or "/".
		//       Needs testing on windows.

		// Calculate the package import path by removing the first matching occurrence
		// of the configured src-path + a slash string.
		pkgPath := strings.Replace(dir, fmt.Sprintf("%v%v", c.Config.SrcPath, string(os.PathSeparator)), "", 1)
		log.WithField("candidate-pkg-path", pkgPath).Debug("Scanning for go package and imports")
		var goPkg *load.Package
		goPkg, err := loadPackageDynamic(c.Config.SrcPath, pkgPath)
		if err != nil {
			pkg.LatestCrawl().AddMessage(fmt.Sprintf("loading %v: %s", pkgPath, err))
			if goPkg == nil {
				return ErrPackageInvalid
			}
			log.WithField("candidate-pkg-path", goPkg.Root).Debugf("Ignoring non-fatal error=%s because still got some data back", err)
		}
		// Skip vendored imports.
		if Vendored(goPkg.ImportPath) {
			return nil
		}

		// TODO: Add function for this to normalize the sub-pkg import path
		// going in.
		subPkg := domain.NewSubPackage(goPkg.Name)
		pc.Data.SubPackages[goPkg.ImportPath] = subPkg

		subPkg.Readme, subPkg.Synopsis = detectReadme(dir)
		// log.Infof("%# v", *goPkg)
		for _, imp := range goPkg.Imports {
			if path, ok := extractVendored(imp); ok {
				imp = path
			}
			if c.Config.IncludeStdLib || strings.Contains(imp, ".") {
				// importsMap[imp] = struct{}{}
				subPkg.Imports = append(subPkg.Imports, imp)
			}
		}
		for _, imp := range goPkg.TestImports {
			if path, ok := extractVendored(imp); ok {
				imp = path
			}
			if c.Config.IncludeStdLib || strings.Contains(imp, ".") {
				// testImportsMap[imp] = struct{}{}
				subPkg.TestImports = append(subPkg.TestImports, imp)
			}
		}
		return nil
	}

	errs := []error{}
	if err := scanDir(localPath); err != nil {
		errs = append(errs, err)
	}

	dirs, err := subDirs(localPath)
	if err != nil {
		// If not an apparent go package, bail out immediately.
		if err == ErrNoGoFiles {
			return err
		}
		errs = append(errs, err)
	}

	for _, dir := range dirs {
		if err := scanDir(dir); err != nil {
			errs = append(errs, err)
		}
	}

	pkg.NormalizeSubPackageKeys()

	if len(errs) > 0 {
		for _, err := range errs {
			pc.AddMessage(err.Error())
		}
	}

	// pc.Data.Imports = []string{}
	// pc.Data.TestImports = []string{}

	// for imp, _ := range importsMap {
	//	pc.Data.Imports = append(pc.Data.Imports, imp)
	// }
	// for imp, _ := range testImportsMap {
	// 	pc.Data.TestImports = append(pc.Data.TestImports, imp)
	// }

	switch strings.ToLower(pkg.VCS) {
	case "git":
		if err = gitStats(pkg.Data, localPath); err != nil {
			return err
		}
	case "hg", "mercurial":
		// TODO: isolate if-condition and make this work for mercurial.
	}

	return nil
}

func gitStats(snap *domain.PackageSnapshot, path string) error {
	if err := sizes(snap, path, ".git"); err != nil {
		return err
	}

	// Number of commits.
	{
		cmd := exec.Command("git", "rev-list", "--count", "HEAD")
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("path", path).Errorf("Counting git commits: %s (output=%v)", err, string(out))
		}
		n, err := strconv.Atoi(strings.Trim(string(out), "\r\n"))
		if err != nil {
			log.WithField("path", path).Errorf("Parsing git commit count: %s (output=%v)", err, string(out))
		}
		snap.Commits = int32(n)
	}

	// Number of branches.
	{
		cmd := exec.Command("git", "branch", "-r")
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("path", path).Errorf("Counting git branches: %s (output=%v)", err, string(out))
			return fmt.Errorf("git stats: counting branches: %s", err)
		}
		snap.Branches = int32(len(strings.Split(strings.Trim(string(out), "\r\n"), "\n")))
	}

	// Number of tags.
	{
		cmd := exec.Command("git", "tag", "-l")
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("path", path).Errorf("Counting git tags: %s (output=%v)", err, string(out))
			return fmt.Errorf("git stats: counting tags: %s", err)
		}
		snap.Branches = int32(len(strings.Split(strings.Trim(string(out), "\r\n"), "\n")))
	}

	// Last commit hash and timestamp.
	{
		const gitLayout = "Mon Jan 2 15:04:05 2006 -0700"
		cmd := exec.Command("git", "log", "-1", "--pretty=format:%H %cd")
		cmd.Dir = path
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.WithField("path", path).Errorf("Getting last info: %s (output=%v)", err, string(out))
			return fmt.Errorf("git stats: last info: %s", err)
		}
		s := strings.Trim(string(out), "\r\n")
		pieces := strings.SplitN(s, " ", 2)
		if len(pieces) < 2 {
			return fmt.Errorf("git stats: received malformed output from log, expected 2 words separated by a space but got: %v", s)
		}
		snap.CommitHash = pieces[0]
		ts, err := time.Parse(gitLayout, pieces[1])
		if err != nil {
			return fmt.Errorf("git stats: unexpected date format, parse failed for: %v", pieces[1])
		}
		snap.CommittedAt = &ts
	}

	return nil
}

func sizes(snap *domain.PackageSnapshot, path string, vcsDir string) error {
	size, err := dirSize(path)
	if err != nil {
		return err
	}
	snap.BytesTotal = size

	if size, err = dirSize(filepath.Join(path, vcsDir)); err != nil {
		return err
	}
	snap.BytesVCS = size
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

// subDirs returns a list of subdirectories under the specified path.
func subDirs(path string) ([]string, error) {
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

func dirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if info != nil {
			if !info.IsDir() {
				size += uint64(info.Size())
			}
		}
		return nil
	})
	return size, err
}

func analyzeFiles(snap *domain.PackageSnapshot, localPath string) error {
	numGoFiles, err := countFiles(localPath, ".go", false)
	if err != nil {
		return err
	}
	if numGoFiles == 0 {
		return ErrNoGoFiles
	}
	snap.NumGoFiles = int32(numGoFiles)

	numFiles, err := countFiles(localPath, "", true)
	if err != nil {
		return err
	}
	snap.NumFiles = int32(numFiles)
	return nil
}

func countFiles(path string, suffix string, includeVendored bool) (int, error) {
	numFiles := 0
	walkFn := func(p string, info os.FileInfo, _ error) error {
		if info == nil || !includeVendored && Vendored(p) {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), suffix) {
			numFiles++
		}
		return nil
	}
	if err := filepath.Walk(path, walkFn); err != nil {
		return numFiles, err
	}
	return numFiles, nil
}

// detectReadme returns (readme content, synopsis).
func detectReadme(localPath string) (string, string) {
	candidates := []string{
		"README.md",
		"Readme.md",
		"readme.md",
		"README.markdown",
		"Readme.markdown",
		"readme.markdown",
		"README.mkd",
		"Readme.mkd",
		"readme.mkd",
		"README",
		"Readme",
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
				if strings.HasSuffix(readmePath, ".md") || strings.HasSuffix(readmePath, ".mkd") || strings.HasSuffix(readmePath, ".markdown") {
					if parse := index.ParseMarkdown(data); parse != nil {
						data = parse.Text
					}
				}
				if len(data) > 25*1024 {
					data = data[:25*1024]
				}
				summary := godoc.Synopsis(string(data))
				if len(summary) > 2048 {
					summary = summary[0:2048]
				}
				return string(data), summary
			}
		}
	}
	return "", ""
}

// PackagePathToRepoRoot isolates and returns a corresponding *vcs.RepoRoot for the
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
func PackagePathToRepoRoot(pkgPath string) (*vcs.RepoRoot, error) {
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
		rr.Repo = strings.Replace(rr.Repo, "https://gitlab.com/", "git@gitlab.com:", 1)
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

// Vendored returns true if the path contains evidence of endoring.
func Vendored(path string) bool {
	v := strings.Contains(path, "/vendor/") || strings.Contains(path, "Godep/_workspace/src/") || strings.Contains(path, "/_vendor/") || strings.Contains(path, "Godeps/_workspace/src/")
	return v
}

func extractVendored(imp string) (path string, ok bool) {
	splitters := []string{
		"/vendor/",
		"Godep/_workspace/src/",
		"/_vendor/",
		"Godeps/_workspace/src/",
		// "Godep/",
		// "Godeps/",
	}
	for _, splitter := range splitters {
		if pieces := strings.SplitN(imp, splitter, 2); len(pieces) > 1 {
			ok = true
			path = pieces[1]
			return
		}
	}
	return
}
