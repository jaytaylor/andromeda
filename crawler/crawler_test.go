package crawler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gigawatt.io/testlib"
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/twilightzone/go/cmd/go/external/cfg"
)

func TestCrawlerRun(t *testing.T) {
	initLog()

	dbFile := filepath.Join(os.TempDir(), testlib.CurrentRunningTest()+".bolt")
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
		if _, err := dbClient.ToCrawlAdd(toCrawls, nil); err != nil {
			return err
		}

		tcl, _ := dbClient.ToCrawlsLen()
		t.Logf("%v", tcl)
		if expected, actual := len(toCrawls), tcl; actual != expected {
			t.Errorf("Expected len(to-crawl entries)=%v but actual=%v", expected, actual)
		}

		// Run the crawler.
		doneCh := make(chan error)
		stopCh := make(chan struct{})
		cfg := NewConfig()
		cfg.IncludeStdLib = true
		cfg.SrcPath = filepath.Join(os.TempDir(), "andromeda-crawler-correctness")
		defer os.RemoveAll(cfg.SrcPath)

		m := NewMaster(dbClient, cfg)
		go func() {
			if err := m.Run(stopCh); err != nil && err != ErrStopRequested {
				doneCh <- err
			}
			doneCh <- nil
		}()

		var (
			lastNumPackages = -1
			waitingSince    = time.Now()
			waitTimeout     = 5 * time.Minute
		)

		func() {
			for {
				select {
				case err := <-doneCh:
					t.Logf("doneCh received result: %s", err)
					if err != nil {
						t.Fatal(err)
					}

				case <-time.After(5 * time.Second):
					if time.Now().After(waitingSince.Add(waitTimeout)) {
						t.Fatalf("Timed out after %s waiting for crawl of runtime package to be accepted", waitTimeout)
					}

					nT, _ := dbClient.ToCrawlsLen()
					nP, _ := dbClient.PackagesLen()
					t.Logf("Checking for crawl result; Queue len=%v; Packages len=%v", nT, nP)
					t.Logf("nP=%v lastNumPackages=%v", nP, lastNumPackages)

					if nP == lastNumPackages {
						t.Logf("Num packages=%v stabilized after %s", nP, time.Now().Sub(waitingSince))
						go func() {
							stopCh <- struct{}{}
							t.Logf("stopCh send finished")
						}()
						time.Sleep(10 * time.Second)
						return
					}
					t.Logf("Num packages=%v, waited %s so far", nP, time.Now().Sub(waitingSince))
					lastNumPackages = nP
				}
			}
		}()

		// Verify results.
		pkg, err := dbClient.Package("runtime")
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := 10, len(pkg.ImportedBy); actual < expected {
			t.Errorf("Expected num imported-by for pkg=runtime >= %v, but actual=%v", expected, actual)
		}

		return nil
	}); err != nil {
		t.Error(err)
	}
}

func TestImportsStd(t *testing.T) {
	initLog()

	goPkg, err := loadPackageDynamic(filepath.Join(cfg.BuildContext.GOPATH, "src"), "testing")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("imports=%v", goPkg.Imports)
	t.Logf("testImports=%v", goPkg.TestImports)
}

func TestImportsNonStd(t *testing.T) {
	initLog()

	var (
		pkgPath   = "andromedadynimporttest"
		parentDir = filepath.Join(os.TempDir(), "src")
		localPath = filepath.Join(parentDir, pkgPath)
	)

	if err := os.RemoveAll(localPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(localPath, os.FileMode(int(0700))); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(localPath); err != nil {
			t.Fatal(err)
		}
	}()

	const goMain = `package main

import (
	"fmt"
)

func main() {
	fmt.Println("hello world!")
}`

	if err := ioutil.WriteFile(filepath.Join(localPath, "main.go"), []byte(goMain), os.FileMode(int(0600))); err != nil {
		t.Fatal(err)
	}

	goPkg, err := loadPackageDynamic(parentDir, pkgPath)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := 1, len(goPkg.Imports); actual != expected {
		t.Errorf("Expected len(goPkg.Imports)=%v but actual=%v", expected, actual)
	}
	if expected, actual := "fmt", goPkg.Imports[0]; actual != expected {
		t.Errorf("Expected goPkg.Imports[0]=%v but actual=%v", expected, actual)
	}

	t.Logf("goPkg=%v", goPkg.Imports)
	t.Logf("testImports=%v", goPkg.TestImports)
}

func TestGitStats(t *testing.T) {
	initLog()

	srcPath := filepath.Join(os.Getenv("GOPATH"), "src", "jaytaylor.com", "andromeda")
	snap := &domain.PackageSnapshot{
		Repo: "git@github.com:jaytaylor/andromeda",
	}
	if err := gitStats(snap, srcPath); err != nil {
		t.Fatal(err)
	}
	t.Logf("snap: %# v", snap)
}

func initLog() {
	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	}
}
