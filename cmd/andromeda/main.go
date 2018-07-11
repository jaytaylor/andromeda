package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
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
	"jaytaylor.com/andromeda/pkg/openssl"
	"jaytaylor.com/andromeda/web"
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&DBFile, "db", "b", DBFile, "Path to BoltDB file")

	rootCmd.AddCommand(bootstrapCmd)

	bootstrapCmd.Flags().StringVarP(&BootstrapGoDocPackagesFile, "godoc-packages-file", "g", "", "Path to local api.godoc.org/packages file to use")
	bootstrapCmd.Flags().IntVarP(&discovery.AddBatchSize, "batch-size", "B", discovery.AddBatchSize, "Batch size per DB transaction when bulk-loading to-crawl entries")
	bootstrapCmd.Flags().StringVarP(&discovery.InputFormat, "format", "f", discovery.InputFormat, "Input format, can be \"json\" or \"text\"")
	bootstrapCmd.Flags().BoolVarP(&discovery.UseXZFileDecompression, "xz", "x", discovery.UseXZFileDecompression, "Activate XZ decompression when reading file-based input (including STDIN)")

	rootCmd.AddCommand(webCmd)

	webCmd.Flags().StringVarP(&WebAddr, "addr", "a", "", "Interface bind address:port spec")

	rootCmd.AddCommand(crawlCmd)

	crawlCmd.Flags().StringVarP(&crawler.DefaultSrcPath, "src-path", "s", crawler.DefaultSrcPath, "Path to checkout source code to")
	crawlCmd.Flags().BoolVarP(&crawler.DefaultDeleteAfter, "delete-after", "d", crawler.DefaultDeleteAfter, "Delete source code after analysis")
	crawlCmd.Flags().IntVarP(&crawler.DefaultMaxItems, "max-items", "m", crawler.DefaultMaxItems, "Maximum number of package items to crawl (<=0 signifies unlimited)")

	rootCmd.AddCommand(localEnqueueCmd)

	localEnqueueCmd.Flags().IntVarP(&db.DefaultQueuePriority, "priority", "p", db.DefaultQueuePriority, "Priority level to use when adding items to the queue")
	localEnqueueCmd.Flags().StringVarP(&EnqueueReason, "reason", "r", EnqueueReason, "Reason to use for to-crawl entries")

	rootCmd.AddCommand(remoteCmd)

	remoteCmd.AddCommand(remoteEnqueueCmd)

	remoteEnqueueCmd.Flags().IntVarP(&db.DefaultQueuePriority, "priority", "p", db.DefaultQueuePriority, "Priority level to use when adding items to the queue")
	remoteEnqueueCmd.Flags().StringVarP(&EnqueueReason, "reason", "r", EnqueueReason, "Reason to use for to-crawl entries")
	addRemoteFlags(remoteEnqueueCmd)

	rootCmd.AddCommand(remoteCrawlerCmd)

	remoteCrawlerCmd.Flags().StringVarP(&crawler.DefaultSrcPath, "src-path", "s", crawler.DefaultSrcPath, "Path to checkout source code to")
	remoteCrawlerCmd.Flags().BoolVarP(&crawler.DefaultDeleteAfter, "delete-after", "d", crawler.DefaultDeleteAfter, "Delete source code after analysis")
	remoteCrawlerCmd.Flags().IntVarP(&crawler.DefaultMaxItems, "max-items", "m", crawler.DefaultMaxItems, "Maximum number of package items to crawl (<=0 signifies unlimited)")

	addRemoteFlags(remoteCrawlerCmd)

	rootCmd.AddCommand(rebuildDBCmd)

	rebuildDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	rootCmd.AddCommand(rebuildAndCleanupDBCmd)

	rebuildAndCleanupDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	rootCmd.AddCommand(queueDeleteCmd)
	rootCmd.AddCommand(repoRootCmd)
	rootCmd.AddCommand(purgeTableCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(checkCmd)
}

var (
	DBFile        = "andromeda.bolt"
	RebuildDBFile string
	Quiet         bool
	Verbose       bool

	BootstrapGoDocPackagesFile string

	EnqueueReason = "requested at cmdline"

	WebAddr string

	CrawlServerAddr = "127.0.01:8082"

	// TODO: Server-side integration not yet complete.  Currently terminating SSL
	//       with Apache.
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string
	AutoTLSCert bool // When true, will use OpenSSL to automatically retrieve the SSL/TLS public key of the gRPC server.
)

func addRemoteFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&CrawlServerAddr, "addr", "a", CrawlServerAddr, "Crawl server host:port address spec")

	cmd.Flags().StringVarP(&TLSCertFile, "cert", "c", TLSCertFile, "SSL/TLS public key certifcate file for mutual CA-based authentication, or in the case of a client connecting over HTTPS, this will be the SSL/TLS public-key certificate file belonging to the SSL-terminating server.")
	cmd.Flags().StringVarP(&TLSKeyFile, "key", "k", TLSKeyFile, "SSL/TLS private key certificate file for mutual TLS CA-based authentication")
	cmd.Flags().StringVarP(&TLSCAFile, "ca", "", TLSCAFile, "ca.crt file for mutual TLS CA-based authentication")
	cmd.Flags().BoolVarP(&AutoTLSCert, "auto-cert", "C", AutoTLSCert, "Use OpenSSL to automatically fill in the SSL/TLS public key of the gRPC server")

}

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

var localEnqueueCmd = &cobra.Command{
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
					Reason:      EnqueueReason,
					SubmittedAt: &now,
				}
			}
			opts := db.NewQueueOptions()
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

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "RPCs",
	Long:  "RPCs",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
}

var remoteEnqueueCmd = &cobra.Command{
	Use:   "enqueue",
	Short: "Add items to the to-crawl queue",
	Long:  "Add items to the to-crawl queue",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			cfg = crawler.NewConfig()
			r   = crawler.NewRemote(CrawlServerAddr, cfg)
		)

		if err := configureRemoteCrawler(r); err != nil {
			log.Fatalf("main: configuring remote crawler: %s", err)
		}

		now := time.Now()
		toCrawls := make([]*domain.ToCrawlEntry, len(args))
		for i, arg := range args {
			toCrawls[i] = &domain.ToCrawlEntry{
				PackagePath: arg,
				Reason:      EnqueueReason,
				SubmittedAt: &now,
			}
		}

		n, err := r.Enqueue(toCrawls, db.DefaultQueuePriority)
		if err != nil {
			log.Fatal(err)
		}
		log.WithField("num-submitted", len(args)).WithField("num-added", n).Info("Enqueue operation successfully completed")
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
	if AutoTLSCert {
		log.WithField("addr", r.Addr).Debug("Automatically downloading SSL/TLS certificate public-key")
		var err error
		if TLSCertFile, err = openssl.DownloadCertPubKey(r.Addr); err != nil {
			return err
		}
	}

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

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check dependencies are installed",
	Long:  "Check that dependencies are installed and within required versions",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			checkGit = func() error {
				// Example version output:
				//     git version 2.17.1

				log.Debug("Checking git version")

				cmd := exec.Command("git", "--version")
				out, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("git: version check command failed: %s", err)
				}
				out = bytes.Trim(out, " \t\r\n")
				log.Debugf("Detected git version: %v", string(out))

				pieces := strings.Split(string(out), " ")
				if len(pieces) < 3 {
					return fmt.Errorf("git: version check command output not recognized, output=%v", string(out))
				}
				versions := strings.Split(pieces[2], ".")
				if len(versions) < 3 {
					return fmt.Errorf("git: version string not recognized, value=%v", pieces[2])
				}
				version := make([]int64, len(versions))
				for i, vs := range versions {
					v, err := strconv.ParseInt(vs, 0, 0)
					if err != nil {
						return fmt.Errorf("git: int parse failed for version component at offset %v for version string %q: %s", pieces[2], err)
					}
					version[i] = v
				}
				if version[0] < 2 {
					return fmt.Errorf("git: major version must be >= 2 but actual installed major version=%v", version[0])
				}
				if version[0] == 2 && version[1] < 3 {
					return fmt.Errorf("git: for 2.x, minor version must be >= 3 but actual installed minor version =%v", version[1])
				}
				return nil
			}

			checkOpenSSL = func() error {
				log.Debug("Checking OpenSSL version")

				cmd := exec.Command("openssl", "version")
				out, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("openssl: version check command failed: %s", err)
				}
				out = bytes.Trim(out, " \t\r\n")
				log.Debugf("Detected openssl version: %v", string(out))
				return nil
			}

			checkFns = []func() error{
				checkGit,
				checkOpenSSL,
			}
			numIssues = 0
		)

		for _, checkFn := range checkFns {
			if err := checkFn(); err != nil {
				log.Error(err)
				numIssues++
			}
		}
		log.Infof("Checks passed: %v / %v", len(checkFns)-numIssues, len(checkFns))
		if numIssues > 0 {
			log.Fatalf("Not all checks passed")
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
