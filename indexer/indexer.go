package indexer

import (
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

type Indexer struct {
	db db.Client
}

func NewIndexer(dbClient db.Client) *Indexer {
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
