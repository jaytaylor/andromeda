package openssl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DownloadCertPubKey Windows implementation.
func DownloadCertPubKey(addr string) (string, error) {
	intermediateFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", fmt.Errorf("creating batch file intermediate temp file: %s", err)
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
		return "", fmt.Errorf("closing batch file intermediate temp file before write: %s", err)
	}
	defer os.Remove(intermediateFile.Name())
	batchFilename := fmt.Sprintf("%v.bat", intermediateFile.Name())
	if err != nil {
		return "", fmt.Errorf("creating batch file: %s", err)
	}
	if err := ioutil.WriteFile(batchFilename, []byte(batchCommands), os.FileMode(int(0600))); err != nil {
		return "", fmt.Errorf("writing batch file %q: %s", batchFilename, err)
	}
	defer os.Remove(batchFilename)

	hostname := strings.Split(addr, ":")[0]
	cmd := exec.Command(batchFilename)
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("downloading certificate public-key: %s", err)
	}
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("%v.pem", hostname))
	if err := ioutil.WriteFile(filename, data, os.FileMode(int(0600))); err != nil {
		return "", fmt.Errorf("writing out PEM file %q: %s", filename, err)
	}
	return filename, nil
}
