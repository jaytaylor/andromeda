package feed

import (
	"errors"
)

var (
	ErrEmptyResult = errors.New("no items in result")
)

// DataSource is the interface definition for feed streams.
type DataSource interface {
	// Update produces a list of new URLs to try to `go get`.
	Refresh() ([]string, error)
}
