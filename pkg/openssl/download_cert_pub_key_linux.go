package openssl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DownloadCertPubKey Linux implementation.
func DownloadCertPubKey(addr string) (string, error) {
	hostname := strings.Split(addr, ":")[0]
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("openssl s_client -showcerts -servername %v -connect %v </dev/null 2>/dev/null | openssl x509 -outform PEM", hostname, addr))
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("downloading certificate public-key: %s", err)
	}
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%v.pem", hostname))
	if err := ioutil.WriteFile(filename, data, os.FileMode(int(0600))); err != nil {
		return "", fmt.Errorf("writing out PEM file %q: %s", filename, err)
	}
	return filename, nil
}
