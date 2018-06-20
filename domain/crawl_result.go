package domain

func NewCrawlResult(pkg *Package, err error) *CrawlResult {
	r := &CrawlResult{
		Package: pkg,
	}
	if err != nil {
		r.Error = err.Error()
	}
	return r
}
