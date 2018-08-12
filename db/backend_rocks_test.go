package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gigawatt.io/testlib"
	"jaytaylor.com/andromeda/domain"
)

func TestRocksBackend(t *testing.T) {
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
		if err := client.be.Put("test0", []byte("hello"), []byte("world")); err != nil {
			t.Error(err)
		}

		if _, err := client.be.Get("test1", []byte("does-not-exist")); err != ErrKeyNotFound {
			t.Errorf("Expected err=%s but actual=%s", ErrKeyNotFound, err)
		}

		if err := client.be.Put("test1", []byte("hello"), []byte("world")); err != nil {
			t.Error(err)
		}

		v, err := client.be.Get("test1", []byte("hello"))
		if err != nil {
			t.Error(err)
		}

		if expected, actual := "world", string(v); actual != expected {
			t.Errorf("Retrieved value did not match inserted value, expected=%v but actual=%v", expected, actual)
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRocksBackendCursor(t *testing.T) {
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

		if err := client.be.View(func(tx Transaction) error {
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
}
