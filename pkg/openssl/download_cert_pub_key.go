package openssl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DownloadCertPubKey retrieves the SSL public key and writes it under /tmp.
func DownloadCertPubKey(addr string) (string, error) {
	data, err := CertPubKey(addr)
	if err != nil {
		log.WithField("addr", addr).Warnf("Problem using openssl to obtain the public-key certificate, falling back to API: %s", err)
		// Fallback to using the public-key certificate API.

		// Specifically helps for cases where the crawler doesn't have openssl
		// installed and available in $PATH, or is operating behind a web proxy.
		url := fmt.Sprintf("https://%[1]v/v1/public-key-crt/%[1]v", addr)
		resp, err := http.Get(url)
		if err != nil {
			return "", fmt.Errorf("downloading from %v: %s", url, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return "", fmt.Errorf("received non-2xx response status-code=%v for url %v", resp.StatusCode, url)
		}
		if data, err = ioutil.ReadAll(resp.Body); err != nil {
			return "", fmt.Errorf("reading response body from url %v: %s", url, err)
		}
	}
	hostname := strings.Split(addr, ":")[0]
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%v.pem", hostname))
	if err := ioutil.WriteFile(filename, data, os.FileMode(int(0600))); err != nil {
		return "", fmt.Errorf("writing out PEM file %q: %s", filename, err)
	}
	return filename, nil
}
