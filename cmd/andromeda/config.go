package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

var (
	// DefaultConfigSearchPaths is a slice of default locations to check for an
	// andromeda TOML configuration file.
	DefaultConfigSearchPaths = []string{
		filepath.Join(os.Getenv("HOME"), ".andromeda.toml"),
		filepath.Join(os.Getenv("HOME"), ".config", "andromeda.toml"),
	}

	ErrNoConfigFileSet = errors.New("no config file specified")
)

// Config is the TOML configuration struct.  When a ~/.andromeda.toml or
// ~/.cfg/andromeda.toml file exists, the values contained therein will
// override the compiled-in defaults.
type Config struct {
	File        string
	SearchPaths []string

	Driver  string
	DB      string
	Quiet   bool
	Verbose bool
}

func NewConfig() *Config {
	cfg := &Config{
		SearchPaths: DefaultConfigSearchPaths,
	}
	return cfg
}

// Do handles discovery (if file hasn't already been set to a non-empty value)
// and parsing of TOML configuration file, then applies new default values.
func (cfg *Config) Do() error {
	if cfg.File == "" {
		if err := cfg.Find(); err != nil {
			return err
		}
	}
	if cfg.File == "" {
		return nil
	}
	if err := cfg.Parse(); err != nil {
		return err
	}
	cfg.Apply()
	return nil
}

// Find searches the default locations for an andromeda TOML configuration file.
//
// If no config file is found, Config.File will not be set and nil is returned.
func (cfg *Config) Find() error {
	for _, path := range cfg.SearchPaths {
		if _, err := os.Stat(path); err == nil {
			log.WithField("path", path).Debug("Located andromeda configuration file")
			cfg.File = path
			return nil
		} else if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

// Parse consumes an andromeda TOML configuration file and sets the values on
// the Config struct instance.
func (cfg *Config) Parse() error {
	if cfg.File == "" {
		return ErrNoConfigFileSet
	}
	if _, err := toml.DecodeFile(cfg.File, &cfg); err != nil {
		return err
	}
	return nil
}

// Apply sets new default values for non-empty / false-y attributes.
func (cfg *Config) Apply() {
	if len(cfg.Driver) > 0 {
		DBDriver = cfg.Driver
	}
	if len(cfg.DB) > 0 {
		DBFile = cfg.DB
	}
	if cfg.Quiet {
		Quiet = true
	}
	if cfg.Verbose {
		Verbose = true
	}
}
