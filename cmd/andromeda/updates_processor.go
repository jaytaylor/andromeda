package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
)

func newUpdatesProcessorCmd() *cobra.Command {
	updatesProcessorCmd := &cobra.Command{
		Use:     "updates-processor",
		Aliases: []string{"updates", "update-processor"},
		Short:   "Andromeda async crawl result updates processor",
		Long:    "Andromeda async crawl result updates processor; makes the most sense when running with postgres as the backing data store",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if MemoryProfiling {
				log.Debug("Starting memory profiler")
				p := profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook)
				defer func() {
					log.Debug("Stopping memory profiler")
					p.Stop()
				}()
			}

			dbCfg := db.NewConfig(DBDriver, DBFile)
			if boltCfg, ok := dbCfg.(*db.BoltConfig); ok {
				boltCfg.BoltOptions.Timeout = 5 * time.Second
			}
			if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
				master := crawler.NewMaster(dbClient, crawler.NewConfig())
				processorStopCh := make(chan struct{})
				go master.SaveLoop(processorStopCh)
				log.Info("Async updates processor started")

				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				select {
				case s := <-sigCh:
					log.WithField("sig", s).Info("Received signal, shutting down updates processor..")
					log.Debug("Stopping updatse processor..")
					processorStopCh <- struct{}{}
				}
				return nil
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	updatesProcessorCmd.Flags().BoolVarP(&MemoryProfiling, "memory-profiling", "", MemoryProfiling, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	return updatesProcessorCmd
}
