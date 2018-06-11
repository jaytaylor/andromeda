package main

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/universe/crawler"
	"jaytaylor.com/universe/db"
	"jaytaylor.com/universe/discovery"
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
	rootCmd.AddCommand(purgePackagesCmd)
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
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()

		if err := bootstrap(cmd, dbClient); err != nil {
			log.Fatalf("main: boostrap: %s", err)
		}

		if err := crawl(cmd, dbClient); err != nil {
			log.Fatalf("main: crawl: %s", err)
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
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()

		if err := bootstrap(cmd, dbClient); err != nil {
			log.Fatalf("main: boostrap: %s", err)
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
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()

		if err := crawl(cmd, dbClient); err != nil {
			log.Fatalf("main: crawl: %s", err)
		}
	},
}

var purgePackagesCmd = &cobra.Command{
	Use:   "purge-packages",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()
		if err := dbClient.Purge(db.TablePackages); err != nil {
			log.Fatalf("main: delete all packages: %s", err)
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
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()

		pl, err := dbClient.PackagesLen()
		if err != nil {
			log.Fatalf("main: getting packages count: %s", err)
		}
		log.WithField("packages", pl).Info("count")

		tcl, err := dbClient.ToCrawlsLen()
		if err != nil {
			log.Fatalf("main: getting to-crawls count: %s", err)
		}
		log.WithField("to-crawls", tcl).Info("count")
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
		dbClient := db.NewClient(db.NewBoltDBConfig(DBFile))
		if err := dbClient.Open(); err != nil {
			log.Fatalf("main: opening db client: %s", err)
		}
		defer func() {
			if err := dbClient.Close(); err != nil {
				log.Fatalf("main: closing db client: %s", err)
			}
		}()

		switch args[0] {
		case db.TablePackages, "package", "pkg":
			pkg, err := dbClient.Package(args[1])
			if err != nil {
				log.Fatalf("main: getting package: %s", err)
			}
			j, err := json.MarshalIndent(pkg, "", "    ")
			if err != nil {
				log.Fatalf("main: marshalling package to JSON: %s", err)
			}
			fmt.Println(string(j))

		case db.TableToCrawl, "to-crawls":
			entry, err := dbClient.ToCrawl(args[1])
			if err != nil {
				log.Fatalf("main: getting to-crawl entry: %s", err)
			}
			j, err := json.MarshalIndent(entry, "", "    ")
			if err != nil {
				log.Fatalf("main: marshalling to-crawl entry to JSON: %s", err)
			}
			fmt.Println(string(j))

		default:
			log.Fatalf("Unrecognized table %q", args[0])
		}
	},
}

func bootstrap(cmd *cobra.Command, dbClient db.DBClient) error {
	cfg := &discovery.BootstrapConfig{
		GoDocPackagesInputFile: BootstrapGoDocPackagesFile,
	}

	return discovery.Bootstrap(dbClient, cfg)
}

func crawl(cmd *cobra.Command, dbClient db.DBClient) error {
	var (
		cfg = &crawler.Config{
			MaxItems: CrawlerMaxItems,
		}
		crawler = crawler.New(dbClient, cfg)
	)

	return crawler.Run()
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
