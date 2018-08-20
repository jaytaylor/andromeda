package db

import (
	"fmt"
	"testing"
)

func TestPostgresBackend(t *testing.T) {
	config := NewPostgresConfig("dbname=andromeda_test host=/var/run/postgresql")

	if err := WithClient(config, func(client Client) error {
		if err := client.Backend().Put(TablePackages, []byte("hello"), []byte("world")); err != nil {
			t.Error(err)
		}

		if _, err := client.Backend().Get(TablePackages, []byte("does-not-exist")); err != ErrKeyNotFound {
			t.Errorf("Expected err=%s but actual=%s", ErrKeyNotFound, err)
		}

		if err := client.Backend().Put(TablePackages, []byte("hello"), []byte("world")); err != nil {
			t.Error(err)
		}

		v, err := client.Backend().Get(TablePackages, []byte("hello"))
		if err != nil {
			t.Error(err)
		}

		if expected, actual := "world", string(v); actual != expected {
			t.Errorf("Retrieved value did not match inserted value, expected=%v but actual=%v", expected, actual)
		}

		// Test iter.
		for _, n := range []int{1, 2, 3} {
			if err := client.Backend().Put(TablePackages, []byte(fmt.Sprintf("hello%v", n)), []byte(fmt.Sprintf("world%v", n))); err != nil {
				t.Error(err)
			}
		}

		tx, err := client.Backend().Begin(false)
		if err != nil {
			t.Fatal(err)
		}
		c := tx.Cursor(TablePackages)
		defer c.Close()
		defer tx.Rollback()
		for k, v := c.First().Data(); c.Err() == nil && len(k) > 0; k, v = c.Next().Data() {
			t.Logf("k=%v v=%v", string(k), string(v))
		}
		if err := c.Err(); err != nil {
			t.Fatal(err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

/*func TestRocksBackendCursor(t *testing.T) {
	var (
		filename = filepath.Join(os.TempDir(), testlib.CurrentRunningTest())
		config   = NewRocksConfig(filename)
	)

	os.RemoveAll(filename)

	defer func() {
		if err := os.RemoveAll(filename); err != nil {
			t.Error(err)
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
				t.Fatalf("[i=%v]%s", i, err)
			}
			l, err := client.PackagesLen()
			if err != nil {
				t.Fatalf("[i=%v] %s", i, err)
			}
			if expected, actual := i+1, l; actual != expected {
				t.Errorf("[i=%v] Expected indexed packages len=%v but actual=%v", i, expected, actual)
			}
		}

		f, err := os.OpenFile("/tmp/jay", os.O_CREATE|os.O_WRONLY, os.FileMode(int(0600)))
		if err != nil {
			t.Fatal(err)
		}

		if err := client.Backend().View(func(tx Transaction) error {
			c := tx.Cursor(TablePackages)
			defer c.Close()

			for k, v := c.First().Data(); len(k) > 0; k, v = c.Next().Data() {
				fmt.Fprintf(f, "k=%v\n", string(k))
				if len(v) > 0 {
					t.Logf("k=%v", string(k))
				}
			}

			return nil
		}); err != nil {
			t.Fatal(err)
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}
}*/
