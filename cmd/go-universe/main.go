package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/jaytaylor/universe/crawler"
	"github.com/jaytaylor/universe/db"
	"github.com/jaytaylor/universe/discovery"
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
