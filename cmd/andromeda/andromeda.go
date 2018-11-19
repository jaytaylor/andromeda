package main

import (
	"encoding/json"
	"fmt"

	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	DBDriver         = "bolt"
	DBFile           = "andromeda.bolt"
	RebuildDBDriver  string
	RebuildDBFile    string
	RebuildDBFilters []string
	Quiet            bool
	Verbose          bool

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

	FeedsEnabled bool

	MemoryProfiler bool
)

func newRootCmd() *cobra.Command {
	configFile := ""

	rootCmd := &cobra.Command{
		Use:   "andromeda",
		Short: "Search the entire visible golang universe",
		Long:  "The most complete golang packages DB in this universe",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogging()
			cfg := NewConfig()
			if configFile != "" {
				cfg.File = configFile
			}
			if err := cfg.Do(); err != nil {
				log.Fatal(err)
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "", "", "Specify andromeda TOML configuration file path (only necessary when deviating from default location of ~/.andromeda.toml or ~/.config/andromeda.toml)")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBDriver, "driver", "", DBDriver, "DB driver backend, one of: bolt, rocks, postgres")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "", DBFile, "Path to DB storage file or directory.  For postgres this must contain the connection string.")

	rootCmd.AddCommand(
		newBootstrapCmd(),
		newWebCmd(),
		newFeedsCmd(),
		newUpdatesProcessorCmd(),
		newLocalCmd(),
		newRemoteCmd(),
		newRemoteCrawlerCmd(),
		newRepoRootCmd(),
		newCheckCmd(),
		newStatsCmd(),
		newServiceCmd(),
		newUtilCmd(),
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
