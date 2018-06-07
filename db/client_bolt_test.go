package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaytaylor/universe/domain"
)

func TestBoltDBClientToCrawlOperations(t *testing.T) {
	filename := filepath.Join(os.TempDir(), "client-tocrawl-test.bolt")

	os.Remove(filename)

	var (
		config = NewBoltDBConfig(filename)
		client = NewClient(config)
	)

	if err := client.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	defer os.Remove(filename)

	pkgNames := []string{
		"foo-bar",
		"jay-tay",
		"want-moar",
	}

	for i, pkgName := range pkgNames {
		now := time.Now()
		entry := &domain.ToCrawlEntry{
			PackageName: pkgName,
			Reason:      "testing",
			SubmittedAt: &now,
		}

		if err := client.ToCrawlAdd(entry); err != nil {
			t.Fatal(err)
		}
		l, err := client.ToCrawlsLen()
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := i+1, l; actual != expected {
			t.Errorf("[i=%v] Expected crawl queue len=%v but actual=%v", i, expected, actual)
		}
	}

	{
		now := time.Now()
		entry := &domain.ToCrawlEntry{
			PackageName: "foo-bar",
			Reason:      "testing2",
			SubmittedAt: &now,
		}
		if err := client.ToCrawlAdd(entry); err != nil {
			t.Fatal(err)
		}
		l, err := client.ToCrawlsLen()
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := 3, l; actual != expected {
			t.Errorf("Expected crawl queue len=%v but actual=%v", expected, actual)
		}
	}

	{
		if err := client.ToCrawlDelete("jay-tay"); err != nil {
			t.Fatal(err)
		}
		l, err := client.ToCrawlsLen()
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := 2, l; actual != expected {
			t.Errorf("Expected crawl queue len=%v but actual=%v", expected, actual)
		}
	}
}

func TestBoltDBClientPackageOperations(t *testing.T) {
	filename := filepath.Join(os.TempDir(), "client-pkg-test.bolt")

	os.Remove(filename)

	var (
		config = NewBoltDBConfig(filename)
		client = NewClient(config)
	)

	if err := client.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	defer os.Remove(filename)

	pkgPaths := []string{
		"jaytaylor.com/archive.is",
		"stdpkg",
		"something/else",
		"github.com/microsoft/github",
		"github.com/pkg/errors",
	}

	for i, pkgPath := range pkgPaths {
		now := time.Now()
		pkg := &domain.Package{
			Path:        pkgPath,
			FirstSeenAt: &now,
		}

		if err := client.PackageSave(pkg); err != nil {
			t.Fatal(err)
		}
		l, err := client.PackagesLen()
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := i+1, l; actual != expected {
			t.Errorf("[i=%v] Expected indexed packages len=%v but actual=%v", i, expected, actual)
		}
	}

	{
		pkg, err := client.Package("jaytaylor.com/archive.is")
		if err != nil {
			t.Error(err)
		}
		if expected, actual := "jaytaylor.com/archive.is", pkg.Path; actual != expected {
			t.Errorf("Expected pkg.Path=%v but actual=%v", expected, actual)
		}
	}

	{
		var (
			n              = 0
			seenArchivePkg bool
		)
		if err := client.Packages(func(pkg *domain.Package) {
			n++
			if !seenArchivePkg && pkg.Path == "jaytaylor.com/archive.is" {
				seenArchivePkg = true
			} else if seenArchivePkg && pkg.Path == "jaytaylor.com/archive.is" {
				t.Fatal("Saw jaytaylor.com/archive.is package more than once during iteration")
			}
		}); err != nil {
			t.Error(err)
		}
		if expected, actual := 5, n; actual != expected {
			t.Errorf("Expected num packages passed through iter func=%v but actual=%v", expected, actual)
		}
		if !seenArchivePkg {
			t.Errorf("Expected to have seen the jaytaylor.com/archive.is package but did not")
		}
	}

	{
		{
			l, err := client.PackagesLen()
			if err != nil {
				t.Fatal(err)
			}
			if expected, actual := 5, l; actual != expected {
				t.Errorf("Expected indexed packages len=%v but actual=%v", expected, actual)
			}
		}

		now := time.Now()
		pkg := &domain.Package{
			Path:        "jaytaylor.com/archive.is",
			FirstSeenAt: &now,
		}
		if err := client.PackageSave(pkg); err != nil {
			t.Fatal(err)
		}

		{
			l, err := client.PackagesLen()
			if err != nil {
				t.Fatal(err)
			}
			if expected, actual := 5, l; actual != expected {
				t.Errorf("Expected indexed packages len=%v but actual=%v", expected, actual)
			}
		}
	}

	{
		if err := client.PackageDelete("jaytaylor.com/archive.is"); err != nil {
			t.Fatal(err)
		}
		l, err := client.PackagesLen()
		if err != nil {
			t.Fatal(err)
		}
		if expected, actual := 4, l; actual != expected {
			t.Errorf("Expected indexed packages len=%v but actual=%v", expected, actual)
		}
	}
}
