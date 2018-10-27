package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/iancoleman/strcase"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

func newLocalCmd() *cobra.Command {
	localCmd := &cobra.Command{
		Use:   "local",
		Short: "Local operations",
		Long:  "Local operations",
		PreRun: func(_ *cobra.Command, args []string) {
			initLogging()
		},
	}

	localCmd.AddCommand(
		newLocalListTablesCmd(),
		newLocalCrawlCmd(),
		newLocalEnqueueCmd(),
		newLocalDeletePackageCmd(),
		newLocalGetCmd(),
		newLocalCatCmd(),
		newLocalDeleteQueueCmd(),
		newLocalDestroyCmd(),
	)

	return localCmd
}

func newLocalListTablesCmd() *cobra.Command {
	crawlCmd := &cobra.Command{
		Use:     "list-tables",
		Aliases: []string{"tables"},
		Short:   `Lists available tables, may optionally specify a type ("key/value" / "kv", or "queue" / "q")`,
		Long:    `Lists available tables, may optionally specify a type ("key/value" / "kv", or "queue" / "q")`,
		PreRun: func(_ *cobra.Command, args []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			var tables []string

			if len(args) > 0 {
				if args[0] == "kv" || args[0] == "key/value" {
					tables = db.KVTables()
				} else if args[0] == "q" || args[0] == "queue" {
					tables = db.QTables()
				} else {
					log.Fatalf(`Unknown table classification type %q; must be one of: "key/value" / "kv", or "queue" / "q"`, args[0])
				}
			} else {
				tables = db.Tables()
			}

			bs, err := json.MarshalIndent(&tables, "", "    ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(bs))
		},
	}

	addCrawlerFlags(crawlCmd)

	return crawlCmd
}

func newLocalCrawlCmd() *cobra.Command {
	crawlCmd := &cobra.Command{
		Use:   "crawl",
		Short: "Local crawl",
		Long:  "Invoke a local crawler for 1 or more packages",
		PreRun: func(_ *cobra.Command, args []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				return crawl(dbClient, args...)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	addCrawlerFlags(crawlCmd)

	return crawlCmd
}

func newLocalEnqueueCmd() *cobra.Command {
	var onlyIfNotExists bool

	localEnqueueCmd := &cobra.Command{
		Use:   "enqueue <package-path-1> [<package-path-2> ...]",
		Short: "Add packages to the to-crawl queue",
		Long:  "Add one or more packages to the to-crawl queue",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
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
				opts.Priority = db.DefaultQueuePriority
				opts.OnlyIfNotExists = onlyIfNotExists

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

	localEnqueueCmd.Flags().IntVarP(&db.DefaultQueuePriority, "priority", "p", db.DefaultQueuePriority, "Priority level to use when adding items to the queue")
	localEnqueueCmd.Flags().StringVarP(&EnqueueReason, "reason", "r", EnqueueReason, "Reason to use for to-crawl entries")
	localEnqueueCmd.Flags().BoolVarP(&onlyIfNotExists, "only-if-not-exists", "", onlyIfNotExists, "Only enqueue if the package does not already exist")

	return localEnqueueCmd
}

func newLocalDeleteQueueCmd() *cobra.Command {
	queueDeleteCmd := &cobra.Command{
		Use:     "queue-delete",
		Aliases: []string{"queue-del", "queue-remove", "queue-rm"},
		Short:   "Remove items from the to-crawl queue",
		Long:    "Remove one or more packages from the to-crawl queue",
		Args:    cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
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
	return queueDeleteCmd
}

func newLocalDestroyCmd() *cobra.Command {
	purgeTablesCmd := &cobra.Command{
		Use:   "destroy <table-or-queue-1> [<table-or-queue-2> ...]",
		Short: "Destroy the named tables and / or queues",
		Long:  "Destroy the named tables and / or queues",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				return dbClient.Destroy(args...)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}
	return purgeTablesCmd
}

func newLocalGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get <table> <key-1> [<key-2> ...]",
		Short: "Retrieve one or more records as JSON based on key from a table as JSON",
		Long:  "Retrieve one or more records as JSON based on key from a table as JSON",
		Args:  cobra.MinimumNArgs(2),
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

			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				t := db.FuzzyTableResolver(args[0])
				if t == "" {
					log.Fatalf("Unrecognized table %q", args[0])
				}

				if len(args) == 2 {
					v, err := dbClient.Backend().Get(t, []byte(args[1]))
					if err != nil {
						log.Fatal(err)
					}
					ptr, err := db.StructFor(t)
					if err != nil {
						log.Fatal(err)
					}
					if err = proto.Unmarshal(v, ptr); err != nil {
						log.Fatal(err)
					}
					return emitJSON(ptr)
				}

				ptrs := []interface{}{}
				for _, arg := range args[1:] {
					v, err := dbClient.Backend().Get(t, []byte(arg))
					if err != nil {
						log.Fatal(err)
					}
					ptr, err := db.StructFor(t)
					if err != nil {
						log.Fatal(err)
					}
					if err = proto.Unmarshal(v, ptr); err != nil {
						log.Fatal(err)
					}
					ptrs = append(ptrs, ptr)
				}
				return emitJSON(ptrs)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	// TODO: use a withMemoryProfiled(func() { ... }) or something to keep this DRY.
	getCmd.Flags().BoolVarP(&MemoryProfiler, "memory-profiler", "", MemoryProfiler, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	return getCmd
}

func newLocalCatCmd() *cobra.Command {
	max := -1

	catCmt := &cobra.Command{
		Use:   "cat <table> [<filter> ...]",
		Short: "Lists table contents with optional filtration; <table> [optional-filter], e.g. pkg -CommittedAt",
		Long:  "Lists table contents with optional filtration; <table> [optional-filter], e.g. pkg -CommittedAt",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				t := db.FuzzyTableResolver(args[0])
				if t == "" {
					log.Fatalf("Unrecognized table %q", args[0])
				}

				// The metadata table is a special case.
				if t == db.TableMetadata {
					m := map[string]string{}
					if err := dbClient.EachRow(db.TableMetadata, func(k []byte, v []byte) {
						m[string(k)] = string(v)
					}); err != nil {
						log.Fatal(err)
					}
					return emitJSON(m)
				}

				if len(args) == 1 {
					return dumpTable(dbClient, t, max)
				}
				return dumpFiltered(dbClient, t, max, args[1])
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	catCmt.Flags().IntVarP(&max, "max", "m", max, "Maximum number of items to include")

	return catCmt
}

// dumpTable dumps both Key/Value and Queue tables as JSON.
func dumpTable(dbClient db.Client, t string, max int) error {
	t = strcase.ToKebab(t)

	fmt.Println("[")
	var (
		i       = 0
		prevPtr interface{}
		iterFn  = func(v []byte) bool {
			ptr, err := db.StructFor(t)
			if err != nil {
				log.Fatal(err)
			}
			if err = proto.Unmarshal(v, ptr); err != nil {
				log.Fatal(err)
			}
			if prevPtr != nil {
				j, _ := json.MarshalIndent(prevPtr, "", "    ")
				fmt.Print(string(j))
				if max <= 0 || max > 1 {
					fmt.Print(",")
				}
				fmt.Print("\n")
			}
			prevPtr = ptr
			i++
			if i >= max {
				return false
			}
			return true
		}
	)

	if db.IsKV(t) {
		if err := dbClient.Backend().EachRowWithBreak(t, func(_ []byte, v []byte) bool {
			return iterFn(v)
		}); err != nil {
			return err
		}
	} else if db.IsQ(t) {
		if err := dbClient.Queue().ScanWithBreak(t, nil, func(v []byte) bool {
			return iterFn(v)
		}); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unknown table %q", t)
	}

	if prevPtr != nil {
		j, _ := json.MarshalIndent(prevPtr, "", "    ")
		fmt.Println(string(j))
	}
	fmt.Println("]")
	return nil
}

// dumpFiltered only supports the packages table.
func dumpFiltered(dbClient db.Client, t string, max int, filter string) error {
	if t != db.TablePackages {
		return errors.New("filtration is only available for the packages table")
	}
	lessFn, err := resolveLessFn(filter)
	if err != nil {
		return err
	}
	h := domain.NewPackagesHeap(lessFn)
	if err := dbClient.EachPackage(func(ptr *domain.Package) {
		h.PackagePush(ptr)
		if max > 0 && h.Len() > max {
			h.PackagePop()
		}
	}); err != nil {
		return err
	}
	return emitJSON(h.Slice())
}

// resolveLessFn maps command-line args to a package field ordering function.
func resolveLessFn(name string) (domain.PackagesLessFunc, error) {
	var (
		orig    = name
		reverse bool
	)
	if strings.HasSuffix(name, "-") {
		name = name[0 : len(name)-1]
		reverse = true
	} else if strings.HasSuffix(name, "+") {
		name = name[0 : len(name)-1]
	}
	name = strings.Replace(strings.Replace(strings.ToLower(name), "-", "", -1), "_", "", -1)
	lessFnMap := map[string]domain.PackagesLessFunc{
		"committedat":   domain.PackagesByCommittedAt,
		"firstseenat":   domain.PackagesByFirstSeenAt,
		"lastseenat":    domain.PackagesByLastSeenAt,
		"bytestotal":    domain.PackagesByBytesTotal,
		"bytesdata":     domain.PackagesByBytesData,
		"bytesvcsdir":   domain.PackagesByBytesVCSDir,
		"numimports":    domain.PackagesByNumImports,
		"numimportedby": domain.PackagesByNumImportedBy,
	}
	lessFn, ok := lessFnMap[name]
	if !ok {
		keys := []string{}
		for k, _ := range lessFnMap {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("less-comparison function %q not found (available funcs: %v)", orig, strings.Join(keys, ", "))
	}
	if reverse {
		lessFn = domain.PackagesDesc(lessFn)
	}
	return lessFn, nil
}

func newLocalDeletePackageCmd() *cobra.Command {
	deletePackageCmd := &cobra.Command{
		Use:     "delete-package <package-path-1> [<package-path-2> ...]",
		Aliases: []string{"delete-packages"},
		Short:   "Delete a package from the database",
		Long:    "Delete a package from the database",
		Args:    cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				return dbClient.PackageDelete(args...)
			}); err != nil {
				log.Fatal(err)
			}
		},
	}
	return deletePackageCmd
}

func addCrawlerFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringVarP(&crawler.DefaultSrcPath, "src-path", "s", crawler.DefaultSrcPath, "Path to checkout source code to")
	cmd.Flags().BoolVarP(&crawler.DefaultDeleteAfter, "delete-after", "d", crawler.DefaultDeleteAfter, "Delete source code after analysis")
	cmd.Flags().IntVarP(&crawler.DefaultMaxItems, "max-items", "m", crawler.DefaultMaxItems, "Maximum number of package items to crawl (<=0 signifies unlimited)")
	cmd.Flags().BoolVarP(&crawler.DefaultEnableGoGit, "go-git", "g", crawler.DefaultEnableGoGit, "Enable golang-native git client instead of relying on executable git binary in $PATH")

	return cmd
}

// crawl performs a local crawl.
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
