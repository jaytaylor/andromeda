package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/universe/crawler"
	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/discovery"
	"jaytaylor.com/universe/domain"
)

var (
	DBFile  = "universe.bolt"
	Quiet   bool
	Verbose bool

	BootstrapGoDocPackagesFile string

	CrawlerMaxItems = -1
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "d", DBFile, "Path to BoltDB file")

	bootstrapCmd.Flags().StringVarP(&BootstrapGoDocPackagesFile, "godoc-packages-file", "g", "", "Path to local api.godoc.org/packages file to use")
	bootstrapCmd.Flags().IntVarP(&discovery.AddBatchSize, "batch-size", "b", discovery.AddBatchSize, "Batch group size per DB transaction")

	crawlCmd.Flags().IntVarP(&CrawlerMaxItems, "max-items", "m", CrawlerMaxItems, "Maximum number of package items to crawl")

	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(crawlCmd)
	rootCmd.AddCommand(purgeTableCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(getCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "go-universe",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	//Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			if err := bootstrap(dbClient); err != nil {
				return fmt.Errorf("boostrap: %s", err)
			}
			if err := crawl(dbClient); err != nil {
				return fmt.Errorf("crawl: %s", err)
			}
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			return bootstrap(dbClient)
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			return crawl(dbClient)
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var purgeTableCmd = &cobra.Command{
	Use:   "purge",
	Short: "[table]",
	Long:  ".. jay will fill this long one out sometime ..",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			switch args[0] {
			case db.TablePackages, "package", "pkg":
				if err := dbClient.Purge(db.TablePackages); err != nil {
					return fmt.Errorf("delete all packages: %s", err)
				}

			case db.TableToCrawl, "to-crawls":
				if err := dbClient.Purge(db.TableToCrawl); err != nil {
					return fmt.Errorf("delete all to-crawl entries: %s", err)
				}

			default:
				return fmt.Errorf("unrecognized table %q", args[0])
			}
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			pl, err := dbClient.PackagesLen()
			if err != nil {
				return fmt.Errorf("getting packages count: %s", err)
			}
			log.WithField("packages", pl).Info("count")

			tcl, err := dbClient.ToCrawlsLen()
			if err != nil {
				return fmt.Errorf("getting to-crawls count: %s", err)
			}
			log.WithField("to-crawls", tcl).Info("count")
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "[table] [key]",
	Long:  ".. jay will fill this long one out sometime ..",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithDBClient(db.NewBoltDBConfig(DBFile), func(dbClient db.DBClient) error {
			switch args[0] {
			case db.TablePackages, "package", "pkg":
				pkg, err := dbClient.Package(args[1])
				if err != nil {
					return fmt.Errorf("getting package: %s", err)
				}
				j, err := json.MarshalIndent(pkg, "", "    ")
				if err != nil {
					return fmt.Errorf("marshalling package to JSON: %s", err)
				}
				fmt.Println(string(j))

			case db.TableToCrawl, "to-crawls":
				var entry *domain.ToCrawlEntry
				if err := dbClient.ToCrawlsWithBreak(func(e *domain.ToCrawlEntry) bool {
					if e.PackagePath == args[1] {
						entry = e
						return false
					}
					return true
				}); err != nil {
					return err
				}
				if entry == nil {
					return fmt.Errorf("to-crawl entry %q not found", args[1])
				}
				j, err := json.MarshalIndent(entry, "", "    ")
				if err != nil {
					return fmt.Errorf("marshalling to-crawl entry to JSON: %s", err)
				}
				fmt.Println(string(j))

			default:
				return fmt.Errorf("unrecognized table %q", args[0])
			}
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

func bootstrap(dbClient db.DBClient) error {
	cfg := &discovery.BootstrapConfig{
		GoDocPackagesInputFile: BootstrapGoDocPackagesFile,
	}

	return discovery.Bootstrap(dbClient, cfg)
}

func crawl(dbClient db.DBClient) error {
	cfg := crawler.NewConfig()
	cfg.MaxItems = CrawlerMaxItems

	stopCh := make(chan struct{})
	errCh := make(chan error)
	c := crawler.New(dbClient, cfg)

	go func() {
		errCh <- c.Run(stopCh)
	}()

	var (
		err      error
		received bool
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err = <-errCh:
	case s := <-sigCh:
		log.WithField("sig", s).Info("Received signal")
		select {
		case stopCh <- struct{}{}:
		case err = <-errCh:
			received = true
		}
		if !received {
			err = <-errCh
		}
	}
	if err != nil && err != crawler.ErrStopRequested {
		return err
	}
	return nil
}

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
