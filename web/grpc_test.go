package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

func initLogging() {
	log.AddHook(filename.NewHook())
	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	}
}

func TestRemote(t *testing.T) {
	initLogging()

	dbFile := filepath.Join(os.TempDir(), "andromeda-test-remote.bolt")
	if err := os.RemoveAll(dbFile); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbFile)

	toCrawls := []*domain.ToCrawlEntry{
		&domain.ToCrawlEntry{PackagePath: "testing"},
		&domain.ToCrawlEntry{PackagePath: "net/http"},
		&domain.ToCrawlEntry{PackagePath: "path/filepath"},
		&domain.ToCrawlEntry{PackagePath: "runtime"},
		&domain.ToCrawlEntry{PackagePath: "os"},
	}

	if err := db.WithClient(db.NewBoltConfig(dbFile), func(dbClient db.Client) error {
		var (
			cfg    = crawler.NewConfig()
			master = crawler.NewMaster(dbClient, cfg)
			wsCfg  = &Config{
				Addr:   "127.0.0.1:0",
				Master: master,
			}
			ws = New(dbClient, wsCfg)
		)

		cfg.IncludeStdLib = true
		cfg.MaxItems = 49

		if err := ws.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := ws.Stop(); err != nil {
				t.Fatal(err)
			}
		}()

		var (
			r      = crawler.NewRemote(ws.Addr().String(), cfg)
			stopCh = make(chan struct{})
			doneCh = make(chan struct{})
		)

		{
			// Add several (3+) to-crawl entries.
			n, err := r.Enqueue(toCrawls, db.DefaultQueuePriority)
			if err != nil {
				t.Fatal(err)
			}
			if expected, actual := len(toCrawls), n; actual != expected {
				t.Errorf("Expected number of items added to queue=%v but actual=%v", expected, actual)
			}
			t.Logf("Added %v/%v item(s) to the queue", n, len(toCrawls))
		}
		{
			// Adding them again should return n=0.
			n, err := r.Enqueue(toCrawls, db.DefaultQueuePriority)
			if err != nil {
				t.Fatal(err)
			}
			if expected, actual := 0, n; actual != expected {
				t.Errorf("Expected redundant enqueue operation n=%v but actual=%v", expected, actual)
			}
		}

		go func() {
			r.Run(stopCh)
			doneCh <- struct{}{}
		}()

		<-doneCh

		// Give the master a moment to finish processing.
		defer time.Sleep(1 * time.Second)

		pl, err := dbClient.PackagesLen()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("packages len=%v", pl)
		if expected, actual := 40, pl; actual < expected {
			t.Errorf("Expected number of packages in db >= %v, but actual=%v", expected, actual)
		}

		tcl, err := dbClient.ToCrawlsLen()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("to-crawls len=%v", tcl)
		if expected, actual := 40, tcl; actual < expected {
			t.Errorf("Expected number of to-crawl entries in db >= %v, but actual=%v", expected, actual)
		}

		return nil
	}); err != nil {
		t.Error(err)
	}
}
