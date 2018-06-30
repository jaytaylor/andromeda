package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/discovery"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/web"
)

var (
	DBFile        = "andromeda.bolt"
	RebuildDBFile string
	Quiet         bool
	Verbose       bool

	BootstrapGoDocPackagesFile string

	WebAddr string

	CrawlServerAddr = "127.0.01:8082"

	EnqueuePriority = db.DefaultQueuePriority
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "b", DBFile, "Path to BoltDB file")

	bootstrapCmd.Flags().StringVarP(&BootstrapGoDocPackagesFile, "godoc-packages-file", "g", "", "Path to local api.godoc.org/packages file to use")
	bootstrapCmd.Flags().IntVarP(&discovery.AddBatchSize, "batch-size", "B", discovery.AddBatchSize, "Batch size per DB transaction when bulk-loading to-crawl entries")
	bootstrapCmd.Flags().BoolVarP(&discovery.UseXZFileDecompression, "xz", "x", discovery.UseXZFileDecompression, "Activate XZ decompression when reading file-based input (including STDIN)")

	webCmd.Flags().StringVarP(&WebAddr, "addr", "a", "", "Interface bind address:port spec")

	crawlCmd.Flags().StringVarP(&crawler.DefaultSrcPath, "src-path", "s", crawler.DefaultSrcPath, "Path to checkout source code to")
	crawlCmd.Flags().BoolVarP(&crawler.DefaultDeleteAfter, "delete-after", "d", crawler.DefaultDeleteAfter, "Delete source code after analysis")
	crawlCmd.Flags().IntVarP(&crawler.DefaultMaxItems, "max-items", "m", crawler.DefaultMaxItems, "Maximum number of package items to crawl (<=0 signifies unlimited)")

	enqueueCmd.Flags().IntVarP(&EnqueuePriority, "priority", "p", EnqueuePriority, "Queue priority level")

	remoteCrawlerCmd.Flags().StringVarP(&crawler.DefaultSrcPath, "src-path", "s", crawler.DefaultSrcPath, "Path to checkout source code to")
	remoteCrawlerCmd.Flags().BoolVarP(&crawler.DefaultDeleteAfter, "delete-after", "d", crawler.DefaultDeleteAfter, "Delete source code after analysis")
	remoteCrawlerCmd.Flags().IntVarP(&crawler.DefaultMaxItems, "max-items", "m", crawler.DefaultMaxItems, "Maximum number of package items to crawl (<=0 signifies unlimited)")

	remoteCrawlerCmd.Flags().StringVarP(&CrawlServerAddr, "addr", "a", CrawlServerAddr, "Crawl server host:port address spec")

	rebuildDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(crawlCmd)
	rootCmd.AddCommand(enqueueCmd)
	rootCmd.AddCommand(queueDeleteCmd)
	rootCmd.AddCommand(repoRootCmd)
	rootCmd.AddCommand(purgeTableCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(remoteCrawlerCmd)
	rootCmd.AddCommand(normalizeSubPackageKeysCmd)
	rootCmd.AddCommand(hostsCmd)
	rootCmd.AddCommand(rebuildDBCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "andromeda",
	Short: ".. jay will fill this out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	//Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("See -h/--help for usage information")
		/*if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			if err := bootstrap(dbClient); err != nil {
				return fmt.Errorf("boostrap: %s", err)
			}
			if err := crawl(dbClient); err != nil {
				return fmt.Errorf("crawl: %s", err)
			}
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}*/
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
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
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
	PreRun: func(_ *cobra.Command, args []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			return crawl(dbClient, args...)
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var enqueueCmd = &cobra.Command{
	Use:   "enqueue",
	Short: "Add packages to the to-crawl queue",
	Long:  "Add one or more packages to the to-crawl queue",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			now := time.Now()
			entries := make([]*domain.ToCrawlEntry, len(args))
			for i, arg := range args {
				entries[i] = &domain.ToCrawlEntry{
					PackagePath: arg,
					Reason:      "added by command-line",
					SubmittedAt: &now,
				}
			}
			opts := db.NewQueueOptions()
			opts.Priority = EnqueuePriority
			n, err := dbClient.ToCrawlAdd(entries, opts)
			if err != nil {
				return err
			}
			log.WithField("added", n).WithField("supplied", len(args)).Info("Enqueue operation finished")
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var queueDeleteCmd = &cobra.Command{
	Use:     "queue-delete",
	Aliases: []string{"queue-del", "queue-remove", "queue-rm"},
	Short:   "Remove items from the to-crawl queue",
	Long:    "Remove one or more packages from the to-crawl queue",
	Args:    cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			n, err := dbClient.ToCrawlRemove(args)
			if err != nil {
				return err
			}
			plural := ""
			if len(args) > 1 {
				plural = "s"
			}
			log.WithField("deleted", n).WithField("supplied", len(args)).Infof("Queue item%s removal operation finished", plural)
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var repoRootCmd = &cobra.Command{
	Use:     "repo-root",
	Aliases: []string{"reporoot", "rr"},
	Short:   "Package repository root lookup",
	Long:    "Administrative utilithy to lookup the repository root for a package",
	Args:    cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		rr, err := crawler.PackagePathToRepoRoot(args[0])
		if err != nil {
			log.Fatalf("%s", err)
		}
		bs, err := json.MarshalIndent(rr, "", "    ")
		if err != nil {
			log.Fatalf("%s", err)
		}
		fmt.Printf("%v\n", string(bs))
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
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
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
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
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
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
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
				if err := dbClient.EachToCrawlWithBreak(func(e *domain.ToCrawlEntry) bool {
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

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "[table]",
	Long:  ".. jay will fill this long one out sometime ..",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			switch args[0] {
			case db.TablePackages, "package", "pkg":
				fmt.Println("[")
				var prevPkg *domain.Package
				if err := dbClient.EachPackage(func(pkg *domain.Package) {
					if prevPkg != nil {
						j, _ := json.MarshalIndent(prevPkg, "", "    ")
						fmt.Printf("%v,", string(j))
					}
					prevPkg = pkg
				}); err != nil {
					return err
				}
				if prevPkg != nil {
					j, _ := json.MarshalIndent(prevPkg, "", "    ")
					fmt.Printf("%v\n", string(j))
				}
				fmt.Println("]")

			case db.TableToCrawl, "to-crawls":
				fmt.Println("[")
				var prevEntry *domain.ToCrawlEntry
				if err := dbClient.EachToCrawl(func(entry *domain.ToCrawlEntry) {
					if prevEntry != nil {
						j, _ := json.MarshalIndent(prevEntry, "", "    ")
						fmt.Printf("%v,\n", j)
					}
					prevEntry = entry
				}); err != nil {
					return err
				}
				if prevEntry != nil {
					j, _ := json.MarshalIndent(prevEntry, "", "    ")
					fmt.Printf("%v\n", string(j))
				}
				fmt.Println("]")

			default:
				return fmt.Errorf("unrecognized table %q", args[0])
			}
			return nil
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var webCmd = &cobra.Command{
	Use:   "web",
	Short: ".. jay will fill this long one out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		dbCfg := db.NewBoltConfig(DBFile)
		dbCfg.BoltOptions.Timeout = 5 * time.Second
		if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
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

var remoteCrawlerCmd = &cobra.Command{
	Use:   "remote-crawler",
	Short: ".. jay will fill this long one out sometime ..",
	Long:  ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			cfg       = crawler.NewConfig()
			r         = crawler.NewRemote(CrawlServerAddr, cfg)
			stopCh    = make(chan struct{})
			runDoneCh = make(chan struct{}, 1)
		)

		go func() {
			r.Run(stopCh)
			runDoneCh <- struct{}{}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		select {
		case s := <-sigCh:
			log.WithField("sig", s).Info("Received signal, shutting down remote crawler")
			stopCh <- struct{}{}
			log.Info("Exiting")
			return

		case <-runDoneCh:
			log.Info("Remote crawler run finished")
			log.Info("Exiting")
			return
		}
	},
}

var normalizeSubPackageKeysCmd = &cobra.Command{
	Use:     "normalize-sub-pkg-keys",
	Aliases: []string{"normalize-subpkg-keys"},
	Short:   ".. jay will fill this long one out sometime ..",
	Long:    ".. jay will fill this long one out sometime ..",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		dbCfg := db.NewBoltConfig(DBFile)
		if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
			return dbClient.NormalizeSubPackageKeys()
		}); err != nil {
			log.Fatal(err)
		}
	},
}

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Hosts stats",
	Long:  "List all hosts and package counts per host",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		dbCfg := db.NewBoltConfig(DBFile)
		if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
			hosts, err := dbClient.Hosts()
			if err != nil {
				log.Fatal(err)
			}
			bs, err := json.MarshalIndent(&hosts, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%v\n", string(bs))
			return nil
		}); err != nil {
			log.Fatal(err)
		}
	},
}

var rebuildDBCmd = &cobra.Command{
	Use:   "rebuild-db",
	Short: "Rebuilds the database",
	Long:  "Rebuilds the entire database",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		dbCfg := db.NewBoltConfig(DBFile)
		if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
			return dbClient.RebuildTo(RebuildDBFile)
		}); err != nil {
			log.Fatal(err)
		}
	},
}

func bootstrap(dbClient db.Client) error {
	cfg := &discovery.BootstrapConfig{
		GoDocPackagesInputFile: BootstrapGoDocPackagesFile,
	}

	return discovery.Bootstrap(dbClient, cfg)
}

func crawl(dbClient db.Client, args ...string) error {
	cfg := crawler.NewConfig()

	stopCh := make(chan struct{})
	errCh := make(chan error)
	m := crawler.NewMaster(dbClient, cfg)

	go func() {
		if len(args) > 0 {
			errCh <- m.Do(stopCh, args...)
		} else {
			// Generic queue crawl.
			errCh <- m.Run(stopCh)
		}
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
		log.WithField("sig", s).Info("Received signal, shutting down crawler")
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
		log.AddHook(filename.NewHook())
		level = log.DebugLevel
	}
	if Quiet {
		level = log.ErrorLevel
	}
	log.SetLevel(level)
}
