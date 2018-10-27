package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// Config is the TOML configuration struct.  When a ~/.andromeda.toml or
// ~/.config/andromeda.toml file exists, the values contained therein will
// override the compiled-in defaults.
type Config struct {
	Driver  string
	DB      string
	Quiet   bool
	Verbose bool
}

// doConfig handles initialization and application of new default values if a
// configuration file is found.
func doConfig() {
	file, err := findConfigFile()
	if err != nil {
		log.Fatalf("locating andromeda TOML configuration: %s", err)
	}

	if len(file) == 0 {
		// No configuration file found.
		return
	}

	config, err := parseConfig(file)
	if err != nil {
		log.Fatalf("parsing andromeda TOML configuration file %q: %s", file, err)
	}

	config.Apply()
}

func (config *Config) Apply() {
	if len(config.Driver) > 0 {
		DBDriver = config.Driver
	}
	if len(config.DB) > 0 {
		DBFile = config.DB
	}
	if config.Quiet {
		Quiet = true
	}
	if config.Verbose {
		Verbose = true
	}
}

// parseConfig parses an andromeda TOML configuration file.
func parseConfig(file string) (*Config, error) {
	config := &Config{}
	if _, err := toml.DecodeFile(file, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// findConfigFile searches for a ~/.andromeda.toml or ~/.config/andromeda.toml
// file (in this order).
//
// If no config file is found, ("", nil) is returned.
func findConfigFile() (string, error) {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".andromeda.toml"),
		filepath.Join(os.Getenv("HOME"), ".config", ".andromeda.toml"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
	}
	return "", nil
}
