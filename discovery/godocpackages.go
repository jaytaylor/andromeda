package discovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// URL points to the location of the godoc.org packages listing.
//
// This information can be used as a kernel to bootstrap golang package
// discovery.
var URL = "https://api.godoc.org/packages"

// GoDocPackages corresponds with the structure of the godoc.org packages JSON
// file.
type GoDocPackages struct {
	Results Results `json:"results"`
}

// Results field item of api.godoc.org/packages data.
type Results []Entry

// Entry field item of api.godoc.org/packages data.
type Entry struct {
	Path        string `json:"path"`
	ImportCount int64  `json:"import_count"`
}

// ParseGoDocPackages parses either godoc.org JSON or newline delimited
// plaintext.
func ParseGoDocPackages(r io.Reader) (*GoDocPackages, error) {
	switch InputFormat {
	case "json", "j":
		return parseGoDocPackagesJSON(r)
	case "text", "txt", "t":
		return parseGoDocPackagesText(r)
	default:
		return nil, fmt.Errorf("unrecognized input format %q", InputFormat)
	}
}

// parseGoDocPackagesJSON parses api.godoc.org/packages JSON data.
func parseGoDocPackagesJSON(r io.Reader) (*GoDocPackages, error) {
	var (
		dec = json.NewDecoder(r)
		gdp GoDocPackages
	)
	if err := dec.Decode(&gdp); err != nil {
		return nil, err
	}
	return &gdp, nil
}

// parseGoDocPackagesText parses newline delimited list of packages.
func parseGoDocPackagesText(r io.Reader) (*GoDocPackages, error) {
	var (
		br  = bufio.NewReader(r)
		gdp = &GoDocPackages{
			Results: []Entry{},
		}
	)
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entry := Entry{
			Path: string(line),
		}
		gdp.Results = append(gdp.Results, entry)
	}
	return gdp, nil
}

// ListGoDocPackages downloads the latest packages listing from
// api.godoc.org/packages.
func ListGoDocPackages() (*GoDocPackages, error) {
	log.Info("Downloading package listing from api.godoc.org")
	resp, err := http.Get(URL)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	gdp, err := ParseGoDocPackages(resp.Body)
	if err != nil {
		return nil, err
	}
	log.WithField("len", len(gdp.Results)).Info("Obtained package listing from api.godoc.org")
	return gdp, nil
}
