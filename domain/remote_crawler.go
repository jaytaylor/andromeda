package domain

import (
	"errors"
)

func NewCrawlResult(pkg *Package, err error) *CrawlResult {
	cr := &CrawlResult{
		Package:           pkg,
		ImportedResources: map[string]*PackageReferences{},
	}
	if err != nil {
		cr.ErrMsg = err.Error()
	}
	return cr
}

func (cr CrawlResult) Error() error {
	if cr.ErrMsg == "" {
		return nil
	}
	return errors.New(cr.ErrMsg)
}
