package feed

import (
	"os"
	"path/filepath"
	"testing"

	"jaytaylor.com/andromeda/db"
)

func TestReddit(t *testing.T) {
	filename := filepath.Join(os.TempDir(), "reddit-feed-test.bolt")

	os.Remove(filename)

	var (
		config = db.NewBoltConfig(filename)
		client = db.NewClient(config)
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

	f := NewReddit(client, "golang")

	possiblePkgs, err := f.Refresh()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("possiblePkgs(%v)=%+v", len(possiblePkgs), possiblePkgs)
}
