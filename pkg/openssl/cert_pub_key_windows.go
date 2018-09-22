package openssl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// CertPubKey Windows implementation.
func CertPubKey(addr string) ([]byte, error) {
	hostname := strings.Split(addr, ":")[0]
	intermediateFile, err := ioutil.TempFile("", "")
	if err != nil {
		return []byte{}, fmt.Errorf("creating batch file intermediate temp file: %s", err)
	}
	batchCmds := fmt.Sprintf(
		`
openssl s_client -showcerts -servername %v -connect %v < nul > %[3]v
openssl x509 -outform PEM < %[3]v`,
		hostname,
		addr,
		intermediateFile.Name(),
	)
	if err = intermediateFile.Close(); err != nil {
		return []byte{}, fmt.Errorf("closing batch file intermediate temp file before write: %s", err)
	}
	defer os.Remove(intermediateFile.Name())
	batchFilename := fmt.Sprintf("%v.bat", intermediateFile.Name())
	if err != nil {
		return []byte{}, fmt.Errorf("creating batch file: %s", err)
	}
	if err := ioutil.WriteFile(batchFilename, []byte(batchCmds), os.FileMode(int(0600))); err != nil {
		return []byte{}, fmt.Errorf("writing batch file %q: %s", batchFilename, err)
	}
	defer os.Remove(batchFilename)

	cmd := exec.Command(batchFilename)
	data, err := cmd.Output()
	if err != nil {
		return []byte{}, fmt.Errorf("downloading certificate public-key: %s", err)
	}
	return data, nil
}
