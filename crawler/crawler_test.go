package crawler

import (
	//"fmt"
	"io/ioutil"
	// "net/http"
	// "net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	// "github.com/AaronO/go-git-http"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/domain"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
)

/*type testHandler struct {
	gitServerURL string
}

func (th *restHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reply := fmt.Sprintf(`<!DOCTYPE html><html><head><meta name="go-import" content="%[1]v git http://%[2]v:%[3]v/git/%[1]v"></head></html>`, r.RequestURI, th.gitServerURL)
	fmt.Fprinf(w, reply)
}*/

func TestCrawlerRunCorrectness(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "universe-crawler-correctness.bolt")
	if err := os.RemoveAll(dbFile); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbFile)

	// t.Fatal("test needs implemented")

	/*gitPath := filepath.Join(os.TempDir(), "testgit")
	if err := os.MkdirAll(gitPath, os.FileMode(int(0700))); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(gitPath)

	gS := httptest.NewServer(githttp.New(gitPath))
	gS.Start()
	defer gS.Close()

	s0 := httptest.NewServer(&testHandler{gitServerPort: gS.URL})
	s0.*/

	// Add several (3+) to-crawl entries.
	if err := db.WithDBClient(db.NewBoltDBConfig(dbFile), func(dbClient db.DBClient) error {
		if _, err := dbClient.ToCrawlAdd(
			&domain.ToCrawlEntry{PackagePath: "testing"},
			&domain.ToCrawlEntry{PackagePath: "net/http"},
			&domain.ToCrawlEntry{PackagePath: "path/filepath"},
			&domain.ToCrawlEntry{PackagePath: "os"},
		); err != nil {
			return err
		}

		tcl, _ := dbClient.ToCrawlsLen()
		t.Logf("%v", tcl)
		if expected, actual := 4, tcl; actual != expected {
			t.Errorf("Expected len(to-crawl entries)=%v but actual=%v", expected, actual)
		}

		cfg := NewConfig()
		cfg.SrcPath = filepath.Join(os.TempDir(), "universe-crawler-correctness")
		defer os.RemoveAll(cfg.SrcPath)

		c := New(dbClient, cfg)
		if err := c.Run(); err != nil {
			t.Error(err)
			return nil
		}

		return nil
	}); err != nil {
		t.Error(err)
	}

	// Find a way to serve up said to-crawl entries so the crawler can `go get` them.

	// Run the crawler.

	// Validate that
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
		pkgPath   = "universedynimporttest"
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
