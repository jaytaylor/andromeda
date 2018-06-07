package main

import (
	"os"
	"time"
	"strings"

	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"

	"github.com/jaytaylor/universe/discovery"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		//errorExit(err)
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "go-universe",
	Short: ".. jay will fill this out sometime .."
	Long: ".. jay will fill this long one out sometime .."
	//Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err != nil {
			errorExit(err)
		}
	},
}

// func errorExit(err interface{}) {
// 	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
// 	os.Exit(1)
// }

func initLogging() {
	level := log.InfoLevel
	if Verbose {
		level = log.DebugLevel
	}
	if Quiet {
		level = log.ErrorLevel
	}
	log.SetLevel(level)
}



/*import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"gigawatt.io/oslib"
	goose "jaytaylor.com/GoOse"
	"jaytaylor.com/archive.is"
)

const NLPWebAddr = "127.0.0.1:8000"

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.162 Safari/537.36"

// NamedEntity represents a named-entity as it relates to a document.
type NamedEntity struct {
	Frequency int    `json:"frequency"` // Number of occurrences in document.
	Entity    string `json:"entity"`    // String content value of entity.
	Label     string `json:"label"`     // Category of named entity.
	POS       string `json:"pos"`       // Part-of-speech.
}

type NamedEntities []NamedEntity

var (
	// Favorites string
	Quiet           bool
	Verbose         bool
	AltNLPWebServer string
	RequestTimeout  time.Duration

	PDFProcessorTimeout = 30 * time.Second
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information for this thing",
	Long:  "All software has versions. This is this the one for thing..",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("jay's hydrator thingamajig")
	},
}

func archiveIsFallback(url string, timeout time.Duration) (map[string]interface{}, []byte, *goose.Article, error) {
	var s string

	snapshots, err := archiveis.Search(url, timeout)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(snapshots) > 0 {
		content, err := download(url, timeout)
		if err != nil {
			return nil, nil, nil, err
		}
		s = string(content)
	}
	//s, err := archiveis.Capture(url)
	//if err != nil {
	//	return nil, nil, nil, err
	//}
	article, err := goose.New().ExtractFromRawHTML(url, s)
	if err != nil {
		return nil, nil, nil, err
	}
	asJSON, err := json.MarshalIndent(*article, "", "    ")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshalling article: %s", err)
	}

	asMap := map[string]interface{}{}
	if err := json.Unmarshal(asJSON, &asMap); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshalling to map: %s", err)
	}
	return asMap, asJSON, article, nil
}

func handlePDF(url string) ([]byte, error) {
	var (
		ch   = make(chan error, 1)
		text []byte
		cmd  = exec.Command("bash", "-c", fmt.Sprintf("set -o errexit && set -o pipefail && set -o nounset && curl -k -sSL -o /tmp/pdf.pdf %q | pdf2htmlEX --auto-hint 1 --correct-text-visibility 1 --process-annotation 1 /tmp/pdf.pdf /tmp/pdf.html", url))
	)

	go func() {
		out, err := cmd.CombinedOutput()
		if err != nil {
			ch <- fmt.Errorf("converting PDF to HTML: %s (output=%v)", err, string(out))
			return
		}
		if text, err = ioutil.ReadFile("/tmp/pdf.html"); err != nil {
			ch <- err
			return
		}
		ch <- nil
	}()

	select {
	case err := <-ch:
		if err != nil {
			return nil, err
		}

	case <-time.After(PDFProcessorTimeout):
		log.Errorf("Timed out after %s processing PDF from %v", PDFProcessorTimeout, url)
		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("Failed to kill PDF converter process: %s", err)
			return nil, fmt.Errorf("killing PDF converter process: %s", err)
		}
		return nil, fmt.Errorf("timed out after %s processing PDF from %v", PDFProcessorTimeout, url)
	}

	return text, nil
}

func withNLPWeb(fn func(baseURL string) error) error {
	var baseURL string

	if AltNLPWebServer == "" {
		nlpSig, nlpAck, err := launchNLPWeb()
		if err != nil {
			return fmt.Errorf("starting nlpweb.py: %s", err)
		}

		defer func() {
			nlpSig <- os.Interrupt
			<-nlpAck
		}()

		baseURL = fmt.Sprintf("http://%v", NLPWebAddr)
	} else {
		baseURL = AltNLPWebServer
	}

	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = fmt.Sprintf("http://%v", baseURL)
	}

	err := fn(baseURL)

	if err != nil {
		return err
	}
	return nil
}

func launchNLPWeb() (chan os.Signal, chan struct{}, error) {
	cmd := exec.Command("/usr/bin/env", "bash", "-c",
		fmt.Sprintf(": && set -o errexit && cd %q && source ../venv/bin/activate && python ../nlpweb.py %v", oslib.PathDirName(os.Args[0]), NLPWebAddr),
	)

	{
		r, w := io.Pipe()
		cmd.Stdout = w
		go func() {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				log.Debugf("[nlpweb][stdout] %v", scanner.Text())
			}
		}()
	}

	{
		r, w := io.Pipe()
		cmd.Stderr = w
		go func() {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				log.Debugf("[nlpweb][stderr] %v", scanner.Text())
			}
		}()
	}

	log.Debugf("Starting nlpweb.py on address=%v", NLPWebAddr)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	var (
		d = net.Dialer{
			Timeout: 1 * time.Second,
		}
		since   = time.Now()
		maxWait = 10 * time.Second
	)
	for {
		if time.Now().Sub(since) > maxWait {
			return nil, nil, fmt.Errorf("timed out after %s waiting for nlpweb.py to start", maxWait)
		}
		conn, err := d.Dial("tcp", NLPWebAddr)
		if err == nil {
			if err = conn.Close(); err != nil {
				return nil, nil, fmt.Errorf("unexpected error closing connection to nlpweb.py: %s", err)
			}
			break
		}
		log.Debug(".")
		time.Sleep(100 * time.Millisecond)
	}
	log.Debugf("Started nlpweb.py OK, pid=%v", cmd.Process.Pid)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, os.Kill)

	ack := make(chan struct{})

	go func() {
		<-sig // Wait for ^C signal.
		fmt.Fprintln(os.Stderr, "\nInterrupt or kill signal detected, shutting down..")

		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("Shutting down nlpweb.py: %s", err)
		}
		log.Debugf("Killed nlpweb.py") // out=%v err=%v", stdout.String(), stderr.String())

		select {
		case ack <- struct{}{}:
		default:
		}
	}()

	return sig, ack, nil
}

func download(url string, timeout time.Duration) ([]byte, error) {
	req, err := newGetRequest(url)
	if err != nil {
		return nil, err
	}

	client := newClient(timeout)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		log.WithField("url", url).WithField("status-code", resp.StatusCode).Error("Received non-2xx response from URL (falling back to archive.is search)")
		// Fallback to archive.is.
		snapshots, err := archiveis.Search(url, timeout)
		log.WithField("url", url).WithField("snapshots", len(snapshots)).Info("Found archive.is snapshots")
		if err == nil && len(snapshots) > 0 {
			if req, err = newGetRequest(snapshots[0].URL); err == nil {
				if resp, err = client.Do(req); err != nil {
					log.WithField("url", snapshots[0].URL).Errorf("Received error from URL: %s", err)
					return nil, fmt.Errorf("even archive.is fallback failed: %s", err)
				}
				if resp.StatusCode/100 != 2 {
					log.WithField("url", snapshots[0].URL).WithField("status-code", resp.StatusCode).Error("Received non-2xx response from URL")
					return nil, fmt.Errorf("even archive.is fallback produced non-2xx response status code=%v", resp.StatusCode)
				}
			}
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body from %v: %s", url, err)
	}
	if err := resp.Body.Close(); err != nil {
		return data, fmt.Errorf("closing body from %v: %s", url, err)
	}

	return data, nil

}

func newGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get request to %v: %s", url, err)
	}

	split := strings.Split(url, "://")
	proto := split[0]
	hostname := strings.Split(split[1], "/")[0]

	req.Header.Set("Host", hostname)
	req.Header.Set("Origin", hostname)
	req.Header.Set("Authority", hostname)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Referer", fmt.Sprintf("%v://%v", proto, hostname))

	return req, nil
}

func newClient(timeout time.Duration) *http.Client {
	c := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout,
			}).Dial,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return c
}*/
