package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/crawler/feed"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

func newFeedsCmd() *cobra.Command {
	feedsCmd := &cobra.Command{
		Use:     "feeds",
		Aliases: []string{"feed"},
		Short:   "Feeds collector",
		Long:    "Stand-alone feeds collector service, can insert directly to the DB or submit over gRPC to an Andromeda web-server",
		//		ArgAliases: []string{}"
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
				f := newFeed(dbClient)
				if err := f.Start(); err != nil {
					log.Fatal(err)
				}

				if WebAddr != "" {
					go runRemoteFeedConnector(WebAddr, f, db.DefaultQueuePriority)
				} else {
					go runDBFeedConnector(dbClient, f, db.DefaultQueuePriority)
				}

				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				select {
				case s := <-sigCh:
					log.WithField("sig", s).Info("Received signal, shutting down feeds monitor..")
					if err := f.Stop(); err != nil {
						log.Fatal(err)
					}
				}
				return nil
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	feedsCmd.Flags().IntVarP(&db.DefaultQueuePriority, "prioerity", "p", db.DefaultQueuePriority, fmt.Sprintf("Prioerity level for queued items; value: 1-%v", db.MaxPriority))
	feedsCmd.Flags().StringVarP(&WebAddr, "addr", "a", WebAddr, "Remote andromeda server address:port (automatically activates Remote mode as opposed to direct DB insertion)")
	feedsCmd.Flags().StringVarP(&feed.DefaultSchedule, "feeds-refresh-schedule", "", feed.DefaultSchedule, "Feeds refresh update cron schedule")
	feedsCmd.Flags().BoolVarP(&MemoryProfiler, "memory-profiler", "", MemoryProfiler, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	return feedsCmd
}

func newFeed(dbClient db.Client) *feed.Feed {
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

func runRemoteFeedConnector(addr string, f *feed.Feed, priority int) {
	var (
		cfg = crawler.NewConfig()
		r   = crawler.NewRemote(addr, cfg)
		b   = newBackoff()
	)
	for {
		ch := f.Channel()
		if ch == nil {
			return
		}
		possiblePkg, ok := <-ch
		if !ok {
			log.Info("Feeds Remote Connector shutting down due to closed feeds channel")
			return
		}
		entry := domain.NewToCrawlEntry(possiblePkg, "discovered by feed crawler")
		for {
			opts := db.NewQueueOptions()
			opts.Priority = priority
			opts.OnlyIfNotExists = true

			n, err := r.Enqueue([]*domain.ToCrawlEntry{entry}, opts)
			if err == nil {
				plural := ""
				if n > 1 || n == 0 {
					plural = "s"
				}
				log.WithField("candidate", possiblePkg).Debugf("Enqueued %v possible pkg%v", n, plural)
				b.Reset()
				break
			}
			d := b.NextBackOff()
			log.WithField("candidate", possiblePkg).Errorf("Problem enqueueing: %s; waiting for %s before retrying", err, d)
			time.Sleep(d)
		}
	}
}

func runDBFeedConnector(dbClient db.Client, f *feed.Feed, priority int) {
	b := newBackoff()
	for {
		ch := f.Channel()
		if ch == nil {
			return
		}
		possiblePkg, ok := <-ch
		if !ok {
			log.Info("Feeds DB Connector shutting down due to closed feeds channel")
			return
		}
		for {
			log.WithField("candidate", possiblePkg).Debug("Enqueueing possible pkg")
			entry := domain.NewToCrawlEntry(possiblePkg, "discovered by feed crawler")
			n, err := dbClient.ToCrawlAdd([]*domain.ToCrawlEntry{entry}, &db.QueueOptions{Priority: priority})
			if err == nil {
				plural := ""
				if n > 1 || n == 0 {
					plural = "s"
				}
				log.WithField("candidate", possiblePkg).Debugf("Enqueued %v possible pkg%v", n, plural)
				b.Reset()
				break
			}
			d := b.NextBackOff()
			log.WithField("candidate", possiblePkg).Errorf("Problem enqueueing: %s; waiting for %s before retrying", err, d)
			time.Sleep(d)
		}
	}
}

func newBackoff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 25 * time.Millisecond
	b.MaxElapsedTime = time.Duration(0)
	b.MaxInterval = 5 * time.Second
	b.Multiplier = 1.5
	b.RandomizationFactor = 0.5
	return b
}
