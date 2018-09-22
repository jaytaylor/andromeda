package openssl

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// CertPubKey Linux implementation.
func CertPubKey(addr string) ([]byte, error) {
	hostname := strings.Split(addr, ":")[0]
	inner := fmt.Sprintf("openssl s_client -showcerts -servername %v -connect %v </dev/null 2>/dev/null | openssl x509 -outform PEM", hostname, addr)
	log.Debugf("openssl command: /bin/bash -c '%v'", inner)
	cmd := exec.Command("/bin/bash", "-c", inner)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, fmt.Errorf("downloading certificate public-key: %s (output=%v)", err, string(data))
	}
	return data, nil
}
