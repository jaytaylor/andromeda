package openssl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestDownloadCertPubKey is the universal tester for the DownloadCertPubKey
// implementations.
func TestDownloadCertPubKey(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer s.Close()

	addr := fmt.Sprint(s.Listener.Addr())
	filename, err := DownloadCertPubKey(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("public-key certificate filename: %v", filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("public-key certificate contents:\n%v", string(data))
	defer os.Remove(filename)
}
