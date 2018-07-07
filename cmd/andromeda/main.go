package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/onrik/logrus/filename"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/discovery"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/web"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "b", DBFile, "Path to BoltDB file")

	bootstrapCmd.Flags().StringVarP(&BootstrapGoDocPackagesFile, "godoc-packages-file", "g", "", "Path to local api.godoc.org/packages file to use")
	bootstrapCmd.Flags().IntVarP(&discovery.AddBatchSize, "batch-size", "B", discovery.AddBatchSize, "Batch size per DB transaction when bulk-loading to-crawl entries")
	bootstrapCmd.Flags().StringVarP(&discovery.InputFormat, "format", "f", discovery.InputFormat, "Input format, can be \"json\" or \"text\"")
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

	remoteCrawlerCmd.Flags().StringVarP(&TLSCertFile, "cert", "c", TLSCertFile, "SSL/TLS public key certifcate file for mutual CA-based authentication, or in the case of a client connecting over HTTPS, this will be the SSL/TLS public-key certificate file belonging to the SSL-terminating server.")
	remoteCrawlerCmd.Flags().StringVarP(&TLSKeyFile, "key", "k", TLSKeyFile, "SSL/TLS private key certificate file for mutual TLS CA-based authentication")
	remoteCrawlerCmd.Flags().StringVarP(&TLSCAFile, "ca", "", TLSCAFile, "ca.crt file for mutual TLS CA-based authentication")

	rebuildDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")
	rebuildAndCleanupDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(crawlCmd)
	rootCmd.AddCommand(enqueueCmd)
	rootCmd.AddCommand(queueDeleteCmd)
	rootCmd.AddCommand(repoRootCmd)
	rootCmd.AddCommand(purgeTableCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(remoteCrawlerCmd)
	rootCmd.AddCommand(rebuildAndCleanupDBCmd)
	rootCmd.AddCommand(rebuildDBCmd)
}

var (
	DBFile        = "andromeda.bolt"
	RebuildDBFile string
	Quiet         bool
	Verbose       bool

	BootstrapGoDocPackagesFile string

	EnqueuePriority = db.DefaultQueuePriority

	WebAddr string

	CrawlServerAddr = "127.0.01:8082"

	// TODO: Server-side integration not yet complete.  Currently terminating SSL
	//       with Apache.
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "andromeda",
	Short: "Search the entire visible golang universe",
	Long:  "The most complete golang packages DB in this universe",
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
		if err := emitJSON(rr); err != nil {
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
				return emitJSON(pkg)

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
				return emitJSON(entry)

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

			case db.TablePendingReferences, "pending-refs", "pending":
				fmt.Println("[")
				var prevEntry *domain.PendingReferences
				if err := dbClient.EachPendingReferences(func(pendingRefs *domain.PendingReferences) {
					if prevEntry != nil {
						j, _ := json.MarshalIndent(prevEntry, "", "    ")
						fmt.Printf("%v,\n", j)
					}
					prevEntry = pendingRefs
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

		if err := configureRemoteCrawler(r); err != nil {
			log.Fatalf("main: configuring remote crawler: %s", err)
		}

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

func configureRemoteCrawler(r *crawler.Remote) error {
	if len(TLSCertFile) == 0 && len(TLSKeyFile) == 0 && len(TLSCAFile) == 0 {
		return nil
	}

	var creds credentials.TransportCredentials
	if len(TLSCAFile) > 0 || len(TLSKeyFile) > 0 {
		// Implies intent to use mutual TLS CA authentication.
		log.WithField("cert", TLSCertFile).WithField("key", TLSKeyFile).WithField("ca", TLSCAFile).Debug("Mutual TLS CA authentication activated")

		if len(TLSCertFile) == 0 || len(TLSKeyFile) == 0 || len(TLSCAFile) == 0 {
			return errors.New("mutual TLS CA authentication required all 3 certificate flags: public.cert, private.key, and ca.cert")
		}

		// Load the client certificates from disk.
		certificate, err := tls.LoadX509KeyPair(TLSCertFile, TLSKeyFile)
		if err != nil {
			return fmt.Errorf("could not load TLS key pair cert=%v key=%v: %s", TLSCertFile, TLSKeyFile, err)
		}

		// Create a certificate pool from the certificate authority.
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(TLSCAFile)
		if err != nil {
			return fmt.Errorf("could not read ca certificate: %s", err)
		}

		// Append the certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return errors.New("failed to append ca certs")
		}

		creds = credentials.NewTLS(&tls.Config{
			// N.B.: ServerName _must_ align with the Common Name (CN) in the certificate.
			ServerName:   strings.Split(CrawlServerAddr, ":")[0],
			Certificates: []tls.Certificate{certificate},
			RootCAs:      certPool,
		})
	} else {
		// Implies intent to use SSL/TLS transport.
		log.WithField("cert", TLSCertFile).Debug("Client SSL/TLS transport activated")

		var err error

		if creds, err = credentials.NewClientTLSFromFile(TLSCertFile, ""); err != nil {
			return err
		}
	}
	r.DialOptions = append(r.DialOptions, grpc.WithTransportCredentials(creds))
	return nil
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

var rebuildAndCleanupDBCmd = &cobra.Command{
	Use:   "rebuild-cleanup-db",
	Short: "Rebuilds the database and cleans up records",
	Long:  "Rebuilds the entire database and applies cleanup filters to the records along the way",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dbCfg = db.NewBoltConfig(DBFile)
			err   error
		)

		filterFn := func(bucket []byte, k []byte, v []byte) ([]byte, []byte) {
			if string(bucket) != db.TablePackages {
				return k, v
			}
			pkg := &domain.Package{}
			if err = proto.Unmarshal(v, pkg); err != nil {
				log.Fatalf("Unexpected problem unmarshaling protobuf for key=%v: %s", string(k), err)
			}

			for subPkgPath, _ := range pkg.Data.SubPackages {
				if strings.Contains(subPkgPath, "/vendor/") || strings.Contains(subPkgPath, "Godep/_workspace/") {
					delete(pkg.Data.SubPackages, subPkgPath)
				}
			}
			pkg.NormalizeSubPackageKeys()

			if v, err = proto.Marshal(pkg); err != nil {
				log.Fatalf("Unexpected problem marshaling protobuf for key=%v: %s", string(k), err)
			}
			return k, v
		}

		if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
			return dbClient.RebuildTo(RebuildDBFile, filterFn)
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

func emitJSON(x interface{}) error {
	bs, err := json.MarshalIndent(x, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", string(bs))
	return nil
}
