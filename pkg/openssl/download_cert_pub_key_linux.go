package openssl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DownloadCertPubKey Linux implementation.
func DownloadCertPubKey(addr string) (string, error) {
	hostname := strings.Split(addr, ":")[0]
	inner := fmt.Sprintf("openssl s_client -showcerts -servername %v -connect %v </dev/null 2>/dev/null | openssl x509 -outform PEM", hostname, addr)
	log.Debugf("openssl command: /bin/bash -c '%v'", inner)
	cmd := exec.Command("/bin/bash", "-c", inner)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("downloading certificate public-key: %s (output=%v)", err, string(data))
	}
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%v.pem", hostname))
	if err := ioutil.WriteFile(filename, data, os.FileMode(int(0600))); err != nil {
		return "", fmt.Errorf("writing out PEM file %q: %s", filename, err)
	}
	return filename, nil
}
