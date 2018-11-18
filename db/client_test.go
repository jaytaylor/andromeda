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
	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/domain"
)

func TestClientToCrawlOperations(t *testing.T) {
	configs := newConfigs()

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			if err := withTestClient(config, func(client Client) error {
				pkgPaths := []string{
					"foo-bar",
					"jay-tay",
					"want-moar",
				}

				{
					l, err := client.PackagesLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := 0, l; actual != expected {
						t.Errorf("%vExpected indexed packages len=%v but actual=%v", typ, 0, actual)
					}
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

				// Test CrawlResult functionality.
				{
					{
						l, err := client.CrawlResultsLen()
						if err != nil {
							t.Fatal(err)
						}
						if expected, actual := 0, l; actual != expected {
							t.Fatalf("Expected initial crawl results len=%v but actual=%v", expected, actual)
						}
					}

					cr := &domain.CrawlResult{
						Package: domain.NewPackage(newFakeRR("jaytaylor.com/andromeda", "jaytaylor.com/andromeda")),
					}

					if err := client.CrawlResultAdd(cr, &QueueOptions{Priority: 5}); err != nil {
						t.Fatal(err)
					}

					{
						l, err := client.CrawlResultsLen()
						if err != nil {
							t.Fatal(err)
						}
						if expected, actual := 1, l; actual != expected {
							t.Fatalf("Expected initial crawl results len=%v but actual=%v", expected, actual)
						}
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
	configs := newConfigs()

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			if err := withTestClient(config, func(client Client) error {
				pkgPaths := []string{
					"jaytaylor.com/archive.is",
					"stdpkg",
					"something/else",
					"github.com/microsoft/github",
					"github.com/pkg/errors",
				}

				{
					l, err := client.PackagesLen()
					if err != nil {
						t.Fatalf("%v%s", typ, err)
					}
					if expected, actual := 0, l; actual != expected {
						t.Errorf("%vExpected indexed packages len=%v but actual=%v", typ, 0, actual)
					}
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
	configs := newConfigs()

	for _, config := range configs {
		func(config Config) {
			typ := fmt.Sprintf("[Config=%T] ", config)

			if err := withTestClient(config, func(client Client) error {
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

func TestClientCrossBackendCopy(t *testing.T) {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		bc       = NewBoltConfig(filename)
		pc       = NewPostgresConfig("dbname=andromeda_test host=/var/run/postgresql")
	)
	if err := withTestClient(bc, func(b Client) error {
		now := time.Now()

		entry := &domain.ToCrawlEntry{
			PackagePath: "foo-bar",
			Reason:      "testing force insert",
			SubmittedAt: &now,
			Force:       true,
		}

		if _, err := b.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, nil); err != nil {
			t.Fatal(err)
		}

		pkgs := []*domain.Package{
			domain.NewPackage(newFakeRR("pkg/a", "pkg/a"), &now),
			domain.NewPackage(newFakeRR("pkg/b", "pkg/b"), &now),
			domain.NewPackage(newFakeRR("pkg/c", "pkg/c"), &now),
			domain.NewPackage(newFakeRR("pkg/d", "pkg/d"), &now),
			domain.NewPackage(newFakeRR("pkg/e", "pkg/e"), &now),
		}
		if err := b.PackageSave(pkgs...); err != nil {
			t.Fatal(err)
		}

		return withTestClient(pc, func(p Client) error {
			// TODO: Expose BE publicly so it can be passed in from another client outside of db pkg.
			if err := b.RebuildTo(p); err != nil {
				t.Fatal(err)
			}

			pL, _ := p.PackagesLen()
			tceL, _ := p.ToCrawlsLen()
			t.Logf("pL=%v tceL=%v", pL, tceL)
			return nil
		})
	}); err != nil {
		t.Fatal(err)
	}
}

func TestKVTables(t *testing.T) {
	kvts := map[string]struct{}{}
	for _, table := range KVTables() {
		kvts[table] = struct{}{}
	}
	for _, qTable := range QTables() {
		if _, ok := kvts[qTable]; ok {
			t.Errorf("queue table found in KV tables list: %s", qTable)
		}
	}
}

func newConfigs() []Config {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		configs  = []Config{
			NewBoltConfig(filename),
			NewPostgresConfig("dbname=andromeda_test host=/var/run/postgresql"),
		}
	)
	if rocksSupportAvailable {
		configs = append(configs, NewRocksConfig(filename))
	}
	return configs
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

func isBolt(c Config) bool {
	return strings.Contains(fmt.Sprintf("%T", c), "Bolt")
}
func isRocks(c Config) bool {
	return strings.Contains(fmt.Sprintf("%T", c), "Rocks")
}
func isPostgres(c Config) bool {
	return strings.Contains(fmt.Sprintf("%T", c), "Postgres")
}

func withTestClient(config Config, fn func(client Client) error) error {
	DefaultBoltQueueFilename = fmt.Sprintf("%v-queue.bolt", testlib.CurrentRunningTest())

	filename := filepath.Join(os.TempDir(), testlib.CurrentRunningTest())

	for _, path := range []string{DefaultBoltQueueFilename, filename} {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removing %q before creating client: %s", path, err)
		}
	}

	return WithClient(config, func(client Client) (err error) {
		typ := fmt.Sprintf("[Config=%T] ", config)

		// Setup/defer cleanup of test remnants.  Depends on config type.
		defer func() {
			for _, path := range []string{DefaultBoltQueueFilename, filename} {
				if rmErr := os.RemoveAll(path); rmErr != nil {
					if err == nil {
						err = fmt.Errorf("%vRemoving %q: %s", typ, path, rmErr)
					} else {
						log.Errorf("%v[TEST] Error removing %q: %s (logging because existing err=%s", typ, path, rmErr, err)
					}
					return
				}
			}
		}()

		if isPostgres(config) {
			reset := func() {
				if destroyErr := client.Queue().Destroy(QTables()...); destroyErr != nil {
					if err == nil {
						err = fmt.Errorf("%vCleaning up test queues: %s", typ, destroyErr)
					} else {
						log.Errorf("%v[TEST] Error cleaning up test queues: %s (logging because existing err=%s)", typ, destroyErr, err)
					}
				}
				if dropErr := client.Backend().Destroy(KVTables()...); dropErr != nil {
					if err == nil {
						err = fmt.Errorf("%vCleaning up test tables: %s", typ, dropErr)
					} else {
						log.Errorf("%v[TEST] Error cleaning up test tables: %s (logging because existing err=%s", typ, dropErr, err)
					}
					return
				}
			}

			reset()
			if err != nil {
				return
			}

			defer func() {
				reset()
			}()
		}

		err = fn(client)
		return
	})
}
