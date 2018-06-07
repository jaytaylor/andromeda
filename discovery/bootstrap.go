package discovery

import (
	log "github.com/sirupsen/logrus"

	"github.com/jaytaylor/universe/db"
	"github.com/jaytaylor/universe/domain"
)

var (
	AddBatchSize = 25000
)

func Bootstrap(dbClient db.DBClient) error {
	gdp, err := ListGoDocPackages()
	if err != nil {
		return err
	}

	batch := make([]*domain.ToCrawlEntry, 0, AddBatchSize)
	for _, result := range gdp.Results {
		entry := domain.NewToCrawlEntry(result.Path, "discovery")

		batch = append(batch, entry)

		if len(batch) == AddBatchSize {
			log.WithField("len", len(batch)).Debug("Adding batch to-crawl entries to DB")
			if err := dbClient.ToCrawlAdd(batch...); err != nil {
				return err
			}
			batch = make([]*domain.ToCrawlEntry, 0, AddBatchSize)
		}
	}
	if len(batch) > 0 {
		log.WithField("len", len(batch)).Debug("Adding batch to-crawl entries to DB")
		if err := dbClient.ToCrawlAdd(batch...); err != nil {
			return err
		}
	}

	return nil
}
