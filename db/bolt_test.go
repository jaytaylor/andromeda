package db

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"jaytaylor.com/andromeda/domain"
)

func TestBoltDBClientToCrawlOperations(t *testing.T) {
	filename := filepath.Join(os.TempDir(), "client-tocrawl-test.bolt")

	os.Remove(filename)

	var (
		config = NewBoltConfig(filename)
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
		"foo-bar",
		"jay-tay",
		"want-moar",
	}

	for i, pkgPath := range pkgPaths {
		now := time.Now()
		entry := &domain.ToCrawlEntry{
			PackagePath: pkgPath,
			Reason:      "testing",
			SubmittedAt: &now,
		}

		if _, err := client.ToCrawlAdd(entry); err != nil {
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
			PackagePath: "foo-bar",
			Reason:      "testing2",
			SubmittedAt: &now,
		}
		if _, err := client.ToCrawlAdd(entry); err != nil {
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
		if _, err := client.ToCrawlDequeue(); err != nil {
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
		config = NewBoltConfig(filename)
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
		if err := client.EachPackage(func(pkg *domain.Package) {
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

	// Test client.Packages(...) multi-get.
	{
		pkgs, err := client.Packages(append([]string{"foo"}, pkgPaths...)...)
		if err != nil {
			t.Error(err)
		} else if expected, actual := len(pkgPaths), len(pkgs); actual != expected {
			t.Errorf("Expected number of packages in multi-get map=%v but actual=%v", expected, actual)
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

	// Test hierarchical pkg resolution 1/2.
	{
		pkg, err := client.Package("jaytaylor.com/archive.is/cmd/archive.is")
		if err != nil {
			t.Error(err)
		} else if expected, actual := "jaytaylor.com/archive.is", pkg.Path; actual != expected {
			t.Errorf("Expected package root=%v but actual=%v", expected, actual)
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

	// Test hierarchical pkg resolution 2/2.
	{
		_, err := client.Package("jaytaylor.com/archive.is/cmd/archive.is")
		if expected, actual := ErrKeyNotFound, err; actual != expected {
			t.Errorf("Expected get of non-existant pkg to return error=%v but actual=%v", expected, actual)
		}
	}
}

func TestBoltDBClientMetaOperations(t *testing.T) {
	filename := filepath.Join(os.TempDir(), "client-metadata-test.bolt")

	os.Remove(filename)

	var (
		config = NewBoltConfig(filename)
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

	k := "foo"

	{
		var v []byte
		if err := client.Meta(k, &v); err != nil {
			t.Fatal(err)
		}
		var nilB []byte
		if expected, actual := nilB, v; !reflect.DeepEqual(actual, expected) {
			t.Errorf("Expected initial metadata value=%v but actual=%v", expected, actual)
		}
	}

	if err := client.MetaSave(k, []byte("bar")); err != nil {
		t.Fatal(err)
	}

	{
		var v []byte
		if err := client.Meta(k, &v); err != nil {
			t.Fatal(err)
		}
		if expected, actual := "bar", string(v); actual != expected {
			t.Errorf("Expected metadata value=%v but actual=%v", expected, actual)
		}
	}

	{
		var v string
		if err := client.Meta(k, &v); err != nil {
			t.Fatal(err)
		}
		if expected, actual := "bar", v; actual != expected {
			t.Errorf("Expected metadata value=%v but actual=%v", expected, actual)
		}
	}

	if err := client.MetaDelete(k); err != nil {
		t.Fatal(err)
	}
}