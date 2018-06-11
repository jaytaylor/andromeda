package crawler

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/twilightzone/go/cmd/go/external/cfg"
)

func TestCrawlerRunCorrectness(t *testing.T) {
	dbFile := filepath.Join(os.TempDir(), "universe-crawler-correctness.bolt")
	if err := os.RemoveAll(dbFile); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dbFile)

	t.Fatal("test needs implemented")

	// Add a bunch of to-crawl entries.

	// Find a way to serve up said to-crawl entries so the crawler can `go get` them.

	// Run the crawler.

	// Validate that
}

func TestImportsStd(t *testing.T) {
	goPkg, err := loadPackageDynamic(filepath.Join(cfg.BuildContext.GOROOT, "src"), "testing")
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
