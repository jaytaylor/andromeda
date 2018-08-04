package db

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"golang.org/x/tools/go/vcs"

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

		if _, err := client.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, nil); err != nil {
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
		if _, err := client.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, nil); err != nil {
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

	// Test insertion of dupes using "force" field.
	{
		now := time.Now()
		entry := &domain.ToCrawlEntry{
			PackagePath: "foo-bar",
			Reason:      "testing force insert",
			SubmittedAt: &now,
			Force:       true,
		}
		expected := 4
		for i := 0; i < 2; i++ {
			if _, err := client.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, nil); err != nil {
				t.Fatal(err)
			}
			l, err := client.ToCrawlsLen()
			if err != nil {
				t.Fatal(err)
			}
			if expected, actual := expected+i, l; actual != expected {
				t.Errorf("[i=%v] Expected crawl queue len=%v but actual=%v", i, expected, actual)
			}
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
		if expected, actual := 4, l; actual != expected {
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
		pkg := domain.NewPackage(newFakeRR(pkgPath, pkgPath), &now)

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
		pkg := domain.NewPackage(newFakeRR("jaytaylor.com/archive.is", "jaytaylor.com/archive.is"), &now)
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

	// RecordImportedBy tests.
	{
		// First, create one of the imported packages to ensure both cases get covered.
		{
			pkg := domain.NewPackage(newFakeRR("github.com/ssor/bom", "github.com/ssor/bom"))
			if err := client.PackageSave(pkg); err != nil {
				t.Errorf("Problem saving package=%v: %s", pkg.Path, err)
			}
		}

		var (
			now       = time.Now()
			rr        = newFakeRR("git@github.com:jaytaylor/html2text", "jaytaylor.com/html2text")
			pkg       = domain.NewPackage(rr)
			resources = map[string]*domain.PackageReferences{
				"github.com/olekukonko/tablewriter": &domain.PackageReferences{
					Refs: []*domain.PackageReference{
						domain.NewPackageReference("github.com/olekukonko/tablewriter", &now),
					},
				},
				"github.com/ssor/bom": &domain.PackageReferences{
					Refs: []*domain.PackageReference{
						domain.NewPackageReference("github.com/ssor/bom", &now),
					},
				},
				"golang.org/x": &domain.PackageReferences{
					Refs: []*domain.PackageReference{
						domain.NewPackageReference("golang.org/x/net/html", &now),
						domain.NewPackageReference("golang.org/x/net/html/atom", &now),
					},
				},
			}
		)

		pkg.Data = &domain.PackageSnapshot{
			SubPackages: map[string]*domain.SubPackage{
				"": &domain.SubPackage{
					Active: true,
					Imports: []string{
						"github.com/olekukonko/tablewriter",
						"github.com/ssor/bom",
						"golang.org/x/net/html",
						"golang.org/x/net/html/atom",
					},
					FirstSeenAt: &now,
					LastSeenAt:  &now,
				},
			},
		}

		if err := client.PackageSave(pkg); err != nil {
			t.Errorf("Saving package failed: %s", err)
		}
		if err := client.RecordImportedBy(pkg, resources); err != nil {
			t.Errorf("RecordImportedBy failed: %s", err)
		}

		if err := client.EachPackage(func(pkg *domain.Package) {
			t.Logf("pkg=%v / %+v", pkg.Path, pkg.ImportedBy)
		}); err != nil {
			t.Error(err)
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

func newFakeRR(repo string, root string) *vcs.RepoRoot {
	rr := &vcs.RepoRoot{
		Repo: repo,
		Root: root,
		VCS: &vcs.Cmd{
			Name: "git",
		},
	}
	return rr
}
