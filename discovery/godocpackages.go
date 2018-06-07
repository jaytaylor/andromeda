package discovery

import (
	"encoding/json"
	"io"
	"net/http"
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

// ParseGoDocPackages parses api.godoc.org/packages JSON data.
func ParseGoDocPackages(r io.Reader) (GoDocPackages, error) {
	var (
		dec = json.NewDecoder(r)
		gdp GoDocPackages
	)
	if err := dec.Decode(&gdp); err != nil {
		return gdp, err
	}
	return gdp, nil
}

// ListGoDocPackages downloads the latest packages listing from
// api.godoc.org/packages.
func ListGoDocPackages() (GoDocPackages, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return GoDocPackages{}, nil
	}
	defer resp.Body.Close()

	gdp, err := ParseGoDocPackages(resp.Body)
	if err != nil {
		return gdp, err
	}
	return gdp, nil
}
