package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"gigawatt.io/concurrency"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/crawler/feed"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/web"
)

func newWebCmd() *cobra.Command {
	var (
		runUpdateProcessor bool
		hostnames          []string
	)

	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Andromeda web server",
		Long:  "Runs the Andromeda HTTP1.1/HTTP2 web frontend and HTTP/2 gRPC server",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if MemoryProfiler {
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
				cfg := &web.Config{
					Addr: WebAddr,
					// TODO: DevMode
					Master:    master,
					Hostnames: hostnames,
				}
				ws := web.New(dbClient, cfg)
				if err := ws.Start(); err != nil {
					return err
				}
				log.Infof("Web service started on %s", ws.Addr())

				updateProcessorStopCh := make(chan struct{})
				if runUpdateProcessor {
					go master.SaveLoop(updateProcessorStopCh)
					log.Info("Async updates processor started")
				}

				var f *feed.Feed
				if FeedsEnabled {
					f = newFeed(dbClient)
					if err := f.Start(); err != nil {
						log.Fatal(err)
					}
					go runDBFeedConnector(dbClient, f, db.DefaultQueuePriority)
				}
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				select {
				case s := <-sigCh:
					log.WithField("sig", s).Info("Received signal, shutting down web service..")
					if runUpdateProcessor {
						log.Debug("Stopping updates processor..")
						updateProcessorStopCh <- struct{}{}
					}
					stopFuncs := []func() error{ws.Stop}
					msg := ""
					if FeedsEnabled {
						stopFuncs = append(stopFuncs, f.Stop)
						msg = " and feeds monitor"
					}
					log.Debugf("Stopping web-server%v..", msg)
					if err := concurrency.MultiGo(stopFuncs...); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	webCmd.Flags().StringVarP(&WebAddr, "addr", "a", "", "Interface bind address:port spec")

	webCmd.Flags().BoolVarP(&FeedsEnabled, "feeds", "", FeedsEnabled, "Enable feed data sources crawler for HN and reddit.com/r/golang")
	webCmd.Flags().StringVarP(&feed.DefaultSchedule, "feeds-refresh-schedule", "", feed.DefaultSchedule, "Feeds refresh update cron schedule")

	webCmd.Flags().BoolVarP(&runUpdateProcessor, "update-processor", "", runUpdateProcessor, "Run update-processor routine as a background task; makes the most sense to turn this on when not using postgres as the backing data store")

	webCmd.Flags().BoolVarP(&MemoryProfiler, "memory-profiler", "", MemoryProfiler, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	webCmd.Flags().StringSliceVarP(&hostnames, "hostnames", "", hostnames, "Server domain name(s) / hostnames to enable CORS policy on and allow public certificate requests for; flag may be comma-delimited or repeated multiple times")

	return webCmd
}
