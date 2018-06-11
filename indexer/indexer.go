package indexer

import (
	log "github.com/sirupsen/logrus"

	"github.com/jaytaylor/universe/db"
	"github.com/jaytaylor/universe/domain"
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
}
