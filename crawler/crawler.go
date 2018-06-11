package crawler

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"github.com/jaytaylor/universe/db"
	"github.com/jaytaylor/universe/domain"
)

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

	i := 0
	for ; c.Config.MaxItems <= 0 || i < c.Config.MaxItems; i++ {
		var entry *domain.ToCrawlEntry
		c.db.ToCrawlsWithBreak(func(e *domain.ToCrawlEntry) bool {
			entry = e
			return false
		})
		if entry == nil {
			// Maybe the queue is cleared.
			break
		}
		log.WithField("entry", fmt.Sprintf("%# v", entry)).Debug("Processing")
		pkg, err := c.do(entry.PackagePath)
		if err != nil {
			log.WithField("i", i).Error("Problem: %s", err)
			return err
		}
		log.Debug("pkg=%s", pkg)
	}
	log.WithField("i", i).Info("Run completed")

	return nil
}

func (c *Crawler) do(pkgPath string) (*domain.Package, error) {
	rr, err := vcs.RepoRootForImportPath(pkgPath, true)
	if err != nil {
		return nil, err
	}
	log.Infof("root=%[1]vrepo=%[2]v\nvcs=%[3]T/%[3]v\n", rr.Root, rr.Repo, rr.VCS)

	return nil, nil
}
