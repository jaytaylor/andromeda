package indexer

import (
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
)

type Indexer struct {
	db db.DBClient
}

func NewIndexer(dbClient db.DBClient) *Indexer {
	indexer := &Indexer{
		db: dbClient,
	}
	return indexer
}

func (indexer *Indexer) Run() error {
	log.Info("Indexer starting run")

	pkg := domain.NewPackage("t-test")
	log.Debug("whatevs %s", pkg)
	return nil
}
