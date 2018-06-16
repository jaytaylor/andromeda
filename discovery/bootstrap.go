package discovery

import (
	"os"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
)

var (
	AddBatchSize = 25000
)

type BootstrapConfig struct {
	GoDocPackagesInputFile string // Optional, may be set to "-" to read from STDIN.
}

func Bootstrap(dbClient db.DBClient, config *BootstrapConfig) error {
	var (
		gdp *GoDocPackages
		err error
	)

	// Obtain packages listing.

	if config.GoDocPackagesInputFile != "" {
		var f *os.File
		if config.GoDocPackagesInputFile == "-" {
			f = os.Stdin
		} else {
			if f, err = os.Open(config.GoDocPackagesInputFile); err != nil {
				return err
			}
			defer f.Close()
		}
		gdp, err = ParseGoDocPackages(f)
	} else {
		gdp, err = ListGoDocPackages()
	}
	if err != nil {
		return err
	}

	// Save to DB.

	var (
		batch        = make([]*domain.ToCrawlEntry, 0, AddBatchSize)
		numAttempted int
		numNew       int
		n            int
	)

	doBatch := func() error {
		if n, err = dbClient.ToCrawlAdd(batch...); err != nil {
			return err
		}
		numNew += n
		numAttempted += len(batch)
		batch = make([]*domain.ToCrawlEntry, 0, AddBatchSize)
		log.WithField("this-batch", len(batch)).WithField("total-attempted", numAttempted).WithField("total-new", numNew).Debug("Added batch of to-crawl entries to DB")
		return nil
	}

	for _, result := range gdp.Results {
		entry := domain.NewToCrawlEntry(result.Path, "discovery")

		batch = append(batch, entry)

		if len(batch) == AddBatchSize {
			if err = doBatch(); err != nil {
				return err
			}
		}
	}
	if len(batch) > 0 {
		if err = doBatch(); err != nil {
			return err
		}
		/*
			log.WithField("this-batch", len(batch)).WithField("total-attempted", numAttempted).WithField("total-new", numNew).Debug("Adding batch of to-crawl entries to DB")
			if n, err = dbClient.ToCrawlAdd(batch...); err != nil {
				return err
			}
			numAttempted += len(batch)
			numNew += n
		*/
	}

	return nil
}
