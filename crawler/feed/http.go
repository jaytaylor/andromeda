package feed

import (
	"io"
	"net/http"
	"time"
)

var (
	UserAgent = "Andromeda_Spider_Bot/1.0 (+http://andromeda.gigawatt.io)"
	// UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.99 Safari/537.36"
	Timeout = 10 * time.Second
)

func doRequest(method string, u string, body io.Reader) (*http.Response, error) {
	req, err := newRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	c := newClient()
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func newRequest(method string, u string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}

func newClient() *http.Client {
	c := &http.Client{
		Timeout: Timeout,
	}
	return c
}
