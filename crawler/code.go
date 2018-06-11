package crawler

/*
import (
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/load"
)

func (c *Crawler) Run() error {
	log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler starting")

	var (
		i     = 0
		err   error
		dbErr error
	)

	for ; c.Config.MaxItems <= 0 || i < c.Config.MaxItems; i++ {
		var pkg *domain.Package

		dbErr = c.db.ToCrawlsWithBreak(func(entry *domain.ToCrawlEntry) bool {
			if c.Config.MaxItems > 0 && i >= c.Config.MaxItems {
				return false
			}
			log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
			pkg, err = c.do(entry.PackagePath)
			if pkg != nil {
				if saveErr := c.db.PackageSave(pkg); saveErr != nil {
					log.WithField("i", i).Errorf("Also encountered error saving package: %s", saveErr)
				}
			}
			//pj, _ := json.Marshal(pkg)
			//log.WithField("pkg", string(pj)).Info("Package crawl finished")
			log.WithField("pkg", pkg).Info("Package crawl finished")
			if err != nil {
				log.WithField("i", i).Errorf("Problem: %s", err)
				return false
			}
			log.Debug("pkg=%s", pkg)
			return false
		})
		if dbErr != nil || err != nil {
			break
		}
		if err = c.db.ToCrawlDelete(pkg.Path); err != nil {
			break
		}
		log.WithField("pkg-path", pkg.Path).Debug("Deleted to-crawl entry")
	}
	if dbErr != nil {
		return dbErr
	}
	if err != nil {
		return err
	}
	log.WithField("i", i).Info("Run completed")

	return nil
}

func (c *Crawler) do(pkgPath string) (*domain.Package, error) {
	pkg, err := c.db.Package(pkgPath)
	if err != nil && err != db.ErrKeyNotFound {
		return nil, err
	}

	rr, err := vcs.RepoRootForImportPath(pkgPath, true) // TODO: Only set true when logging is verbose.
	if err != nil {
		return nil, err
	}
	log.Infof("root=%[1]v\nrepo=%[2]v\nvcs=%[3]T/%[3]v\n", rr.Root, rr.Repo, rr.VCS.Name)

	localPath := filepath.Join(c.Config.SrcPath, rr.Root)

	startedAt := time.Now()

	// If first crawl.
	if pkg == nil {
		now := time.Now()
		pkg = &domain.Package{
			FirstSeenAt: &now,
			Path:        rr.Root,
			Name:        "", // TODO: package name???
			URL:         rr.Repo,
			VCS:         rr.VCS.Name,
			Data:        &domain.PackageSnapshot{},
			History:     []*domain.PackageCrawl{},
		}
	}

	defer os.RemoveAll(localPath)

	if err := c.get(rr); err != nil {
		finishedAt := time.Now()
		pc := &domain.PackageCrawl{
			JobMessages:   []string{fmt.Sprintf("go getting: %v", err.Error())},
			JobSucceeded:  false,
			JobStartedAt:  &startedAt,
			JobFinishedAt: &finishedAt,
		}
		pkg.History = append(pkg.History, pc)
		return pkg, err
	}

	pc, err := c.interrogate(rr)

	finishedAt := time.Now()

	pc.JobStartedAt = &startedAt
	pc.JobFinishedAt = &finishedAt

	// TODO: Merge in latest info.
	pkg.History = append(pkg.History, pc)

	if err != nil {
		pc.JobMessages = []string{fmt.Sprintf("interrogating: %v", err.Error())}
		pc.JobSucceeded = false
		return pkg, err
	}

	pkg.Data.Merge(pc.Data)
	pc.JobSucceeded = true
	return pkg, nil
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

func (c *Crawler) interrogate(rr *vcs.RepoRoot) (*domain.PackageCrawl, error) {
	localPath := filepath.Join(c.Config.SrcPath, rr.Root)

	pc := &domain.PackageCrawl{
		Data: &domain.PackageSnapshot{
			Repo: rr.Repo,
		},
		JobMessages: []string{},
	}

	pc.Data.Readme = detectReadme(localPath)

	dirs, err := subdirs(localPath)
	if err != nil {
		return pc, err
	}
	importsMap := map[string]struct{}{}
	testImportsMap := map[string]struct{}{}

	scanDir := func(dir string) {
		pkgPath := strings.Replace(dir, fmt.Sprintf("%v%v", c.Config.SrcPath, os.PathSeparator), "", 1)
		goPkg, err := loadPackageDynamic(c.Config.SrcPath, pkgPath)
		if err != nil {
			pc.JobMessages = append(pc.JobMessages, fmt.Sprintf("loading %v: %s", pkgPath, err))
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
		return pc, err
	}
	pc.Data.Bytes = size

	return pc, nil
}

// loadPackageDynamic returns slices containing imports and test imports.
func loadPackageDynamic(parentDir string, pkgPath string) (*load.Package, error) {
	cfg.GOROOTsrc = parentDir
	cfg.BuildContext.GOROOT = filepath.Dir(parentDir)

	cfg.Gopath = filepath.SplitList(cfg.BuildContext.GOPATH + ":" + parentDir)
	//defer func() { cfg.GOROOTsrc = GOROOTsrcBackup }()

	lps := load.Packages([]string{pkgPath})
	for _, lp := range lps {
		return lp, nil
	}
	return nil, fmt.Errorf("no pkg found")
}

func subdirs(path string) ([]string, error) {
	dirs := []string{}
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
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
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			size += info.Size()
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
*/
