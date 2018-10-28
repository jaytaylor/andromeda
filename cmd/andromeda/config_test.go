package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	// Note: quiet and verbose cannot both be set to true, this is only done here
	// to test that the settings are applied.
	const content = `
driver = "postgres"
db = "dbname=andromeda host=/var/run/postgresql"
quiet = true
verbose = true
`

	tempDir, err := ioutil.TempDir("", "andromeda-config")
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(tempDir, "andromeda.toml")

	DefaultConfigSearchPaths = append([]string{file}, DefaultConfigSearchPaths...)

	if err := ioutil.WriteFile(file, []byte(content), os.FileMode(int(0600))); err != nil {
		t.Fatal(err)
	}

	var (
		origDriver  = DBDriver
		origDB      = DBFile
		origQuiet   = Quiet
		origVerbose = Verbose
	)

	cfg := NewConfig()

	if err := cfg.Do(); err != nil {
		t.Fatal(err)
	}

	if cfg.File != file {
		t.Errorf("Expected cfg.File=%v but actual=%v", file, cfg.File)
	}

	if DBDriver == origDriver {
		t.Errorf("DBDriver value did not change")
	}
	if DBFile == origDB {
		t.Errorf("DBFile value did not change")
	}
	if Quiet == origQuiet {
		t.Errorf("Quiet value did not change")
	}
	if Verbose == origVerbose {
		t.Errorf("Verbose value did not change")
	}

	if expected, actual := "postgres", DBDriver; actual != expected {
		t.Errorf("Expected DBDriver=%v but actual=%v", expected, actual)
	}
	if expected, actual := "dbname=andromeda host=/var/run/postgresql", DBFile; actual != expected {
		t.Errorf("Expected DBFile=%v but actual=%v", expected, actual)
	}
	if expected, actual := true, Quiet; actual != expected {
		t.Errorf("Expected Quiet=%v but actual=%v", expected, actual)
	}
	if expected, actual := true, Verbose; actual != expected {
		t.Errorf("Expected Verbose=%v but actual=%v", expected, actual)
	}
}
