package crawler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/twilightzone/go/cmd/go/external/cfg"
)

func init() {
	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	}
}

func TestCrawlerRun(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "andromeda-crawler-correctness.bolt")
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

		tcl, _ := dbClient.ToCrawlsLen()
		t.Logf("%v", tcl)
		if expected, actual := len(toCrawls), tcl; actual != expected {
			t.Errorf("Expected len(to-crawl entries)=%v but actual=%v", expected, actual)
		}

		// Run the crawler.
		cfg := NewConfig()
		cfg.IncludeStdLib = true
		cfg.SrcPath = filepath.Join(os.TempDir(), "andromeda-crawler-correctness")
		defer os.RemoveAll(cfg.SrcPath)

		m := NewMaster(dbClient, cfg)
		if err := m.Run(nil); err != nil {
			t.Error(err)
			return nil
		}

		// Verify results.
		{
			pkg, err := dbClient.Package("runtime")
			if err != nil {
				return err
			}
			if expected, actual := 10, len(pkg.ImportedBy); actual < expected {
				j, _ := json.MarshalIndent(pkg, "", "    ")
				t.Logf("\"runtime\" package JSON:\n%v", string(j))
				return fmt.Errorf("Expected \"runtime\" to be imported by > %v others but actual=%v", expected, actual)
			}
		}

		return nil
	}); err != nil {
		t.Error(err)
	}
}

func TestImportsStd(t *testing.T) {
	goPkg, err := loadPackageDynamic(filepath.Join(cfg.BuildContext.GOPATH, "src"), "testing")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("imports=%v", goPkg.Imports)
	t.Logf("testImports=%v", goPkg.TestImports)
}

func TestImportsNonStd(t *testing.T) {
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
