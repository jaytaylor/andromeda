package crawler

import (
	//"bytes"
	"fmt"
	"path/filepath"
	//"os/exec"

	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/load"
)

//var GOROOTsrcBackup string

//func init() {
//	GOROOTsrcBackup = cfg.GOROOTsrc
//}

type Config struct {
	MaxItems int // Maximum number of items to process.
}

// TODO: Suppor for picking up where last run left off.

type Crawler struct {
	Config *Config
	db     db.DBClient
}

// New creates and returns a new crawler instance with the supplied db client
// and configuration.
func New(dbClient db.DBClient, cfg *Config) *Crawler {
	c := &Crawler{
		Config: cfg,
		db:     dbClient,
	}
	return c
}

func (c *Crawler) Run() error {
	log.WithField("cfg", fmt.Sprintf("%# v", c.Config)).Info("Crawler starting")

	var (
		i   = 0
		err error
	)

	c.db.ToCrawlsWithBreak(func(entry *domain.ToCrawlEntry) bool {
		if c.Config.MaxItems > 0 && i >= c.Config.MaxItems {
			return false
		}
		i++
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		var pkg *domain.Package
		if pkg, err = c.do(entry.PackagePath); err != nil {
			log.WithField("i", i).Error("Problem: %s", err)
			return false
		}
		log.Debug("pkg=%s", pkg)
		return true
	})

	if err != nil {
		return err

	}
	log.WithField("i", i).Info("Run completed")

	return nil
}

func (c *Crawler) do(pkgPath string) (*domain.Package, error) {
	rr, err := vcs.RepoRootForImportPath(pkgPath, true)
	if err != nil {
		return nil, err
	}
	log.Infof("root=%[1]vrepo=%[2]v\nvcs=%[3]T/%[3]v\n", rr.Root, rr.Repo, rr.VCS.Name)

	return nil, nil
}

// bashSafePrefix activates bash error modes.
const bashSafePrefix = `: ; set -o errexit ; set -o pipefail ; set -o nounset ; `

// imports returns slices containing imports and test imports.
func imports(parentDir string, pkgPath string) ([]string, []string, error) {
	/*var (
		importsCmd = exec.Command("/bin/bash", "-c", fmt.Sprintf(`%vcd %q && go list -f '{{ .Imports }}' . | sed 's/^.\(.*\).$/\1/' | tr ' ' $'\n' | (grep '^[^\/]\+\.[^\/]\+\/.\+' || :)`, bashSafePrefix, localPath))
		out        []byte
		err        error
	)

	if out, err = importsCmd.Output(); err != nil {
		return nil, fmt.Errorf("listing go imports for path %q: %s", localPath, err)
	}

	for _, dep := range bytes.Split(out, []byte{'\n'}) {
		verbose := log.GetLevel() == log.DebugLevel
		repoRoot, err := vcs.RepoRootForImportPath(string(dep), verbose)
		if err != nil {
			log.WithField("import", string(dep)).Errorf("go-import to git URL resolution failed: %s", err)
			continue
		}

		if exists := cb.Context.Get(repoRoot.Repo); exists != nil {
			deps = append(deps, exists)
		} else {
			discovered := domain.NewCodebase(repoRoot.Repo, repoRoot.Repo, cb.Context)
			discovered.Name = string(dep)
			deps = append(deps, discovered)
		}
	}*/

	cfg.GOROOTsrc = parentDir
	cfg.BuildContext.GOROOT = parentDir

	cfg.Gopath = filepath.SplitList(cfg.BuildContext.GOPATH + ":" + parentDir)
	//defer func() { cfg.GOROOTsrc = GOROOTsrcBackup }()

	lps := load.Packages([]string{pkgPath})
	for _, lp := range lps {
		return lp.Imports, lp.TestImports, nil
	}
	return nil, nil, nil
}
