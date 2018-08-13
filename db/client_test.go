package db

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gigawatt.io/testlib"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/domain"
)

func TestClientToCrawlOperations(t *testing.T) {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		configs  = []Config{
			NewBoltConfig(filename),
			NewRocksConfig(filename),
		}
	)

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			os.RemoveAll(filename)

			defer func() {
				if err := os.RemoveAll(filename); err != nil {
					t.Errorf("%v%s", typ, err)
				}
			}()

			if err := WithClient(config, func(client *Client) error {
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
						t.Fatalf("%v%s", typ, err)
					}
					l, err := client.ToCrawlsLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := i+1, l; actual != expected {
						t.Errorf("%v[i=%v] Expected crawl queue len=%v but actual=%v", typ, i, expected, actual)
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
						t.Fatalf("%v%s", typ, err)
					}
					l, err := client.ToCrawlsLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := 3, l; actual != expected {
						t.Errorf("%vExpected crawl queue len=%v but actual=%v", typ, expected, actual)
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
							t.Fatalf("%v%s", typ, err)
						}
						l, err := client.ToCrawlsLen()
						if err != nil {
							t.Fatalf("%v%s", typ, err)
						}
						if expected, actual := expected+i, l; actual != expected {
							t.Errorf("%v[i=%v] Expected crawl queue len=%v but actual=%v", typ, i, expected, actual)
						}
					}
				}

				{
					if _, err := client.ToCrawlDequeue(); err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					l, err := client.ToCrawlsLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := 4, l; actual != expected {
						t.Errorf("%vExpected crawl queue len=%v but actual=%v", typ, expected, actual)
					}
				}
				return nil
			}); err != nil {
				t.Errorf("%v%s", typ, err)
			}
		}(config)
	}
}

func TestClientPackageOperations(t *testing.T) {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		configs  = []Config{
			NewBoltConfig(filename),
			NewRocksConfig(filename),
		}
	)

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			os.RemoveAll(filename)

			defer func() {
				if err := os.RemoveAll(filename); err != nil {
					t.Errorf("%v%s", typ, err)
				}
			}()

			if err := WithClient(config, func(client *Client) error {
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
						t.Fatalf("%v[i=%v]%s", typ, i, err)
					}
					l, err := client.PackagesLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := i+1, l; actual != expected {
						t.Errorf("%v[i=%v] Expected indexed packages len=%v but actual=%v", typ, i, expected, actual)
					}
				}

				{
					pkg, err := client.Package("jaytaylor.com/archive.is")
					if err != nil {
						t.Errorf("%v%s", typ, err)
					}
					if expected, actual := "jaytaylor.com/archive.is", pkg.Path; actual != expected {
						t.Errorf("%vExpected pkg.Path=%v but actual=%v", typ, expected, actual)
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
							t.Fatalf("%vSaw jaytaylor.com/archive.is package more than once during iteration", typ)
						}
					}); err != nil {
						t.Errorf("%v%s", typ, err)
					}
					if expected, actual := 5, n; actual != expected {
						t.Errorf("%vExpected num packages passed through iter func=%v but actual=%v", typ, expected, actual)
					}
					if !seenArchivePkg {
						t.Errorf("%vExpected to have seen the jaytaylor.com/archive.is package but did not", typ)
					}
				}

				// Test client.Packages(...) multi-get.
				{
					pkgs, err := client.Packages(append([]string{"foo"}, pkgPaths...)...)
					if err != nil {
						t.Errorf("%v%s", typ, err)
					} else if expected, actual := len(pkgPaths), len(pkgs); actual != expected {
						t.Errorf("%vExpected number of packages in multi-get map=%v but actual=%v", typ, expected, actual)
					}
				}

				{
					{
						l, err := client.PackagesLen()
						if err != nil {
							t.Fatalf("%v%s", typ, err)
						}
						if expected, actual := 5, l; actual != expected {
							t.Errorf("%vExpected indexed packages len=%v but actual=%v", typ, expected, actual)
						}
					}

					now := time.Now()
					pkg := domain.NewPackage(newFakeRR("jaytaylor.com/archive.is", "jaytaylor.com/archive.is"), &now)
					if err := client.PackageSave(pkg); err != nil {
						t.Fatalf("%v%s", typ, err)
					}

					// Skip this test for RocksDB because counts are imprecise.
					if !isRocks(config) {
						l, err := client.PackagesLen()
						if err != nil {
							t.Fatalf("%v%s", typ, err)
						}
						if expected, actual := 5, l; actual != expected {
							t.Errorf("%vExpected indexed packages len=%v but actual=%v", typ, expected, actual)
						}
					}
				}

				// Test hierarchical pkg resolution 1/2.
				{
					pkg, err := client.Package("jaytaylor.com/archive.is/cmd/archive.is")
					if err != nil {
						t.Errorf("%v%s", typ, err)
					} else if expected, actual := "jaytaylor.com/archive.is", pkg.Path; actual != expected {
						t.Errorf("%vExpected package root=%v but actual=%v", typ, expected, actual)
					}
				}

				{
					if err := client.PackageDelete("jaytaylor.com/archive.is"); err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					l, err := client.PackagesLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if !isRocks(config) {
						if expected, actual := 4, l; actual != expected {
							t.Errorf("%vExpected indexed packages len=%v but actual=%v", typ, expected, actual)
						}
					}
				}

				// Test hierarchical pkg resolution 2/2.
				{
					pkg, err := client.Package("jaytaylor.com/archive.is/cmd/archive.is")
					if expected, actual := ErrKeyNotFound, err; actual != expected {
						t.Fatalf("%vExpected get of non-existant pkg to return error=%v but actual=%v (pkg=%v)", typ, expected, actual, pkg)
					}
				}

				// RecordImportedBy tests.
				{
					// First, create one of the imported packages to ensure both cases get covered.
					{
						pkg := domain.NewPackage(newFakeRR("github.com/ssor/bom", "github.com/ssor/bom"))
						if err := client.PackageSave(pkg); err != nil {
							t.Errorf("%vProblem saving package=%v: %s", typ, pkg.Path, err)
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
						t.Errorf("%vSaving package failed: %s", typ, err)
					}
					if err := client.RecordImportedBy(pkg, resources); err != nil {
						t.Errorf("%vRecordImportedBy failed: %s", typ, err)
					}

					if err := client.EachPackage(func(pkg *domain.Package) {
						// t.Logf("%vEachPackage: current pkg=%v / %+v", typ, pkg.Path, pkg.ImportedBy)
					}); err != nil {
						t.Errorf("%v%s", typ, err)
					}
				}
				return nil
			}); err != nil {
				t.Errorf("%v%s", typ, err)
			}
		}(config)
	}
}

func TestClientMetaOperations(t *testing.T) {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		configs  = []Config{
			NewBoltConfig(filename),
			NewRocksConfig(filename),
		}
	)

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			os.RemoveAll(filename)

			defer func() {
				if err := os.RemoveAll(filename); err != nil {
					t.Errorf("%v%s", typ, err)
				}
			}()

			if err := WithClient(config, func(client *Client) error {
				k := "foo"

				{
					var v []byte
					if err := client.Meta(k, &v); err != ErrKeyNotFound {
						t.Fatalf("%vExpected err=%s but actual=%s", typ, ErrKeyNotFound, err)
					}
					var nilB []byte
					if expected, actual := nilB, v; !reflect.DeepEqual(actual, expected) {
						t.Errorf("%vExpected initial metadata value=%v but actual=%v", typ, expected, actual)
					}
				}

				if err := client.MetaSave(k, []byte("bar")); err != nil {
					t.Fatalf("%v%s", typ, err)
				}

				{
					var v []byte
					if err := client.Meta(k, &v); err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := "bar", string(v); actual != expected {
						t.Errorf("%vExpected metadata value=%v but actual=%v", typ, expected, actual)
					}
				}

				{
					var v string
					if err := client.Meta(k, &v); err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := "bar", v; actual != expected {
						t.Errorf("%vExpected metadata value=%v but actual=%v", typ, expected, actual)
					}
				}

				if err := client.MetaDelete(k); err != nil {
					t.Fatalf("%v%s", typ, err)
				}
				return nil
			}); err != nil {
				t.Errorf("%v%s", typ, err)
			}
		}(config)
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

func isRocks(c Config) bool {
	return strings.Contains(fmt.Sprintf("%T", c), "Rocks")
}
