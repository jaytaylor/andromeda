package discovery

import (
	"strings"
	"testing"
)

func TestParseGoDocPackages(t *testing.T) {
	const s = `{"results":[{"path":"github.com/getgauge/common","import_count":0},{"path":"github.com/jeanfric/goembed","import_count":0},{"path":"github.com/golang/go/src/cmd/api","import_count":0},{"path":"github.com/golang/go/src/cmd/fix","import_count":0},{"path":"github.com/golang/go/src/cmd/cgo","import_count":0},{"path":"github.com/golang/go/src/cmd/vet","import_count":0},{"path":"github.com/awakesecurity/turnpike","import_count":0},{"path":"github.com/couchbase/cbauth","import_count":0},{"path":"github.com/issue9/mux","import_count":0},{"path":"github.com/issue9/assert","import_count":0},{"path":"github.com/bhenderson/web","import_count":0},{"path":"github.com/travis-ci/worker","import_count":0},{"path":"github.com/pasiukevich/inmemory","import_count":0},{"path":"github.com/ORBAT/krater","import_count":0},{"path":"github.com/alindeman/lint2hub","import_count":0},{"path":"github.com/cespare/permute","import_count":0}]}`

	r := strings.NewReader(s)

	gdp, err := ParseGoDocPackages(r)
	if err != nil {
		t.Fatal(err)
	}
	if expected, actual := 16, len(gdp.Results); actual != expected {
		t.Errorf("Expected number of parsed GoDocPackages=%v but actual=%v", expected, actual)
	}
}
