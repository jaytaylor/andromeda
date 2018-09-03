package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
		newLocalCrawlCmd(),
		newLocalEnqueueCmd(),
		newLocalDeletePackageCmd(),
		newLocalGetCmd(),
		newLocalListCmd(),
		newLocalDeleteQueueCmd(),
		newLocalDestroyCmd(),
	)

	return localCmd
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
				/*switch args[0] {
				case db.TablePackages, "package", "pkg":
					if err := dbClient.Purge(db.TablePackages); err != nil {
						return fmt.Errorf("delete all packages: %s", err)
					}

				case db.TableToCrawl, "to-crawls":
					if err := dbClient.Purge(db.TableToCrawl); err != nil {
						return fmt.Errorf("delete all to-crawl entries: %s", err)
					}

				case db.TableMetadata, "metadata", "meta":
					if err := dbClient.Purge(db.TableMetadata); err != nil {
						return fmt.Errorf("delete all metadata entries: %s", err)
					}

				default:
					return fmt.Errorf("unrecognized table %q", args[0])
				}
				return nil*/
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}
	return purgeTablesCmd
}

func newLocalGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "[table] [key]",
		Long:  ".. jay will fill this long one out sometime ..",
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

				case db.TablePendingReferences, "pending-refs", "pending":
					refs, err := dbClient.PendingReferences(args[1])
					if err != nil {
						log.Fatal(err)
					}
					return emitJSON(refs)

				case db.TableMetadata, "metadata", "meta":
					var s string
					if err := dbClient.Meta(args[1], &s); err != nil {
						log.Fatal(err)
					}
					fmt.Printf("%v\n", s)

				default:
					return fmt.Errorf("unrecognized table %q", args[0])
				}
				return nil
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	// TODO: use a withMemoryProfiled(func() { ... }) or something to keep this DRY.
	getCmd.Flags().BoolVarP(&MemoryProfiler, "memory-profiler", "", MemoryProfiler, "Enable the memory profiler; creates a mem.pprof file while the application is shutting down")

	return getCmd
}

func newLocalListCmd() *cobra.Command {
	max := -1

	lsCmd := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"cat"},
		Short:   "[table] [optional-filter]",
		Long:    "[table] [optional-filter], e.g. pkg -CommittedAt",
		Args:    cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				switch args[0] {
				case db.TablePackages, "package", "pkg":
					if len(args) == 1 {
						fmt.Println("[")
						var (
							i       = 0
							prevPkg *domain.Package
						)
						if err := dbClient.EachPackageWithBreak(func(pkg *domain.Package) bool {
							if prevPkg != nil {
								j, _ := json.MarshalIndent(prevPkg, "", "    ")
								fmt.Printf("%v", string(j))
								if max <= 0 || max > 1 {
									fmt.Print(",")
								}
								fmt.Print("\n")
							}
							prevPkg = pkg
							i++
							if i >= max {
								return false
							}
							return true
						}); err != nil {
							return err
						}
						if prevPkg != nil {
							j, _ := json.MarshalIndent(prevPkg, "", "    ")
							fmt.Printf("%v\n", string(j))
						}
						fmt.Println("]")
					} else {
						lessFn, err := resolveLessFn(args[1])
						if err != nil {
							return err
						}
						h := domain.NewPackagesHeap(lessFn)
						if err := dbClient.EachPackage(func(pkg *domain.Package) {
							h.PackagePush(pkg)
							if max > 0 && h.Len() > max {
								h.PackagePop()
							}
						}); err != nil {
							return err
						}
						emitJSON(h.Slice())
					}

				case db.TableToCrawl, "to-crawls":
					fmt.Println("[")
					var prevEntry *domain.ToCrawlEntry
					if err := dbClient.EachToCrawl(func(entry *domain.ToCrawlEntry) {
						if prevEntry != nil {
							j, _ := json.MarshalIndent(prevEntry, "", "    ")
							fmt.Printf("%v", string(j))
							if max <= 0 || max > 1 {
								fmt.Print(",")
							}
							fmt.Print("\n")
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
							fmt.Printf("%v", string(j))
							if max <= 0 || max > 1 {
								fmt.Print(",")
							}
							fmt.Print("\n")
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

				case db.TableMetadata, "metadata", "meta":
					m := map[string]string{}
					if err := dbClient.EachRow(db.TableMetadata, func(k []byte, v []byte) {
						m[string(k)] = string(v)
					}); err != nil {
						log.Fatal(err)
					}
					emitJSON(m)

				default:
					return fmt.Errorf("unrecognized table %q", args[0])
				}
				return nil
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	lsCmd.Flags().IntVarP(&max, "max", "m", max, "Maximum number of items to include")

	return lsCmd
}

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
