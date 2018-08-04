package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoltBackend(t *testing.T) {
	var (
		fileName = filepath.Join(os.TempDir(), "TestBoltBackend.bolt")
		cfg      = NewBoltConfig(fileName)
		be       = NewBoltBackend(cfg)
	)

	os.Remove(fileName)

	if err := be.Open(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := be.Close(); err != nil {
			t.Error(err)
		}
		if err := os.Remove(fileName); err != nil {
			t.Error(err)
		}
	}()

	if _, err := be.Get("test1", []byte("does-not-exist")); err != ErrKeyNotFound {
		t.Errorf("Expected err=%s but actual=%s", ErrKeyNotFound, err)
	}

	if err := be.Put("test1", []byte("hello"), []byte("world")); err != nil {
		t.Error(err)
	}

	v, err := be.Get("test1", []byte("hello"))
	if err != nil {
		t.Error(err)
	}

	if expected, actual := "world", string(v); actual != expected {
		t.Errorf("Retrieved value did not match inserted value, expected=%v but actual=%v", expected, actual)
	}
}
