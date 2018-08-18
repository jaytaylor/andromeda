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
	"jaytaylor.com/andromeda/crawler/feed"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/web"
)

func newWebCmd() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Andromeda web server",
		Long:  "Runs the Andromeda HTTP1.1/HTTP2 web frontend and HTTP/2 gRPC server",
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
			if err := db.WithClient(dbCfg, func(dbClient *db.Client) error {
				master := crawler.NewMaster(dbClient, crawler.NewConfig())
				cfg := &web.Config{
					Addr: WebAddr,
					// TODO: DevMode
					Master: master,
				}
				ws := web.New(dbClient, cfg)
				if err := ws.Start(); err != nil {
					return err
				}
				log.Infof("Web service started on %s", ws.Addr())
				if FeedsEnabled {
					f := newFeed(dbClient)
					if err := f.Start(); err != nil {
						log.Fatal(err)
					}
					defer func() {
						if err := f.Stop(); err != nil {
							log.Error(err)
						}
					}()
					go runFeedConnector(dbClient, f)
				}
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				select {
				case s := <-sigCh:
					log.WithField("sig", s).Info("Received signal, shutting down web service..")
					if err := ws.Stop(); err != nil {
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
	webCmd.Flags().BoolVarP(&MemoryProfiling, "memory-profiling", "", MemoryProfiling, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	return webCmd
}

func newFeed(dbClient *db.Client) *feed.Feed {
	cfg := feed.Config{
		Sources: []feed.DataSource{
			feed.NewHackerNews(dbClient),
			feed.NewReddit(dbClient, "golang"),
		},
		Schedule: feed.DefaultSchedule,
	}
	f := feed.New(cfg)
	return f
}

func runFeedConnector(dbClient *db.Client, f *feed.Feed) {
	for {
		ch := f.Channel()
		if ch == nil {
			return
		}
		possiblePkg, ok := <-ch
		if !ok {
			return
		}
		log.WithField("candidate", possiblePkg).Debug("Enqueueing possible pkg")
		entry := domain.NewToCrawlEntry(possiblePkg, "discovered by feed crawler")
		dbClient.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, nil)
	}
}
