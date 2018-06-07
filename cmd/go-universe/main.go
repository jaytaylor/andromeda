package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/jaytaylor/universe/db"
	"github.com/jaytaylor/universe/discovery"
)

var (
	DBFile  = "universe.bolt"
	Quiet   bool
	Verbose bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "d", DBFile, "Path to BoltDB file")
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

		if err := discovery.Bootstrap(dbClient); err != nil {
			log.Fatalf("main: bootstrap: %s", err)
		}
	},
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
