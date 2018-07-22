package main

import (
	"encoding/json"
	"fmt"

	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	DBFile        = "andromeda.bolt"
	RebuildDBFile string
	Quiet         bool
	Verbose       bool

	BootstrapGoDocPackagesFile string

	EnqueueReason = "requested at cmdline"
	EnqueueForce  bool

	WebAddr string

	CrawlServerAddr = "127.0.01:8082"

	// TODO: Server-side integration not yet complete.  Currently terminating SSL
	//       with Apache.
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string
	AutoTLSCert bool // When true, will use OpenSSL to automatically retrieve the SSL/TLS public key of the gRPC server.

	FeedsEnabled = true

	MemoryProfiling bool
)

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "andromeda",
		Short: "Search the entire visible golang universe",
		Long:  "The most complete golang packages DB in this universe",
	}

	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "b", DBFile, "Path to BoltDB file")

	rootCmd.AddCommand(
		newBootstrapCmd(),
		newWebCmd(),
		newLocalCmd(),
		newRemoteCmd(),
		newRemoteCrawlerCmd(),
		newRebuildDBCmd(),
		newRebuildAndCleanupDBCmd(),
		newRepoRootCmd(),
		newCheckCmd(),
		newStatsCmd(),
		newServiceCmd(),
	)

	return rootCmd
}

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func initLogging() {
	level := log.InfoLevel
	if Verbose {
		log.AddHook(filename.NewHook())
		level = log.DebugLevel
	}
	if Quiet {
		level = log.ErrorLevel
	}
	log.SetLevel(level)
}

func emitJSON(x interface{}) error {
	bs, err := json.MarshalIndent(x, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", string(bs))
	return nil
}
