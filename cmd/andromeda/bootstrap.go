package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/discovery"
)

func newBootstrapCmd() *cobra.Command {
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap a fresh andromeda database",
		Long:  "Bootstrap a fresh andromeda database",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
				return bootstrap(dbClient)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	bootstrapCmd.Flags().StringVarP(&BootstrapGoDocPackagesFile, "godoc-packages-file", "g", "", "Path to local api.godoc.org/packages file to use")
	bootstrapCmd.Flags().IntVarP(&discovery.AddBatchSize, "batch-size", "B", discovery.AddBatchSize, "Batch size per DB transaction when bulk-loading to-crawl entries")
	bootstrapCmd.Flags().StringVarP(&discovery.InputFormat, "format", "f", discovery.InputFormat, "Input format, can be \"json\" or \"text\"")
	bootstrapCmd.Flags().BoolVarP(&discovery.UseXZFileDecompression, "xz", "x", discovery.UseXZFileDecompression, "Activate XZ decompression when reading file-based input (including STDIN)")

	return bootstrapCmd
}

func bootstrap(dbClient db.Client) error {
	cfg := &discovery.BootstrapConfig{
		GoDocPackagesInputFile: BootstrapGoDocPackagesFile,
	}

	return discovery.Bootstrap(dbClient, cfg)
}
