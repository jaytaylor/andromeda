package web

import (
	"os"
	"path/filepath"
	"testing"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

func TestRemote(t *testing.T) {
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
		// Add several (3+) to-crawl entries.
		if _, err := dbClient.ToCrawlAdd(toCrawls...); err != nil {
			return err
		}

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
		cfg.MaxItems = 3

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

		go func() {
			r.Run(stopCh)
			doneCh <- struct{}{}
		}()

		<-doneCh

		pl, err := dbClient.PackagesLen()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("packages len=%v", pl)
		if expected, actual := 40, pl; actual < expected {
			t.Errorf("Expected number of packages in db >= %v, but actual=%v")
		}

		tcl, err := dbClient.ToCrawlsLen()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("to-crawls len=%v", tcl)
		if expected, actual := 40, tcl; actual < expected {
			t.Errorf("Expected number of to-crawl entries in db >= %v, but actual=%v")
		}

		return nil
	}); err != nil {
		t.Error(err)
	}
}
