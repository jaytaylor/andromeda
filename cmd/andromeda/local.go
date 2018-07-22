package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		newCrawlCmd(),
		newLocalEnqueueCmd(),
		newDeletePackageCmd(),
		newGetCmd(),
		newLsCmd(),
		newQueueDeleteCmd(),
		newPurgeTableCmd(),
	)

	return localCmd
}

func newCrawlCmd() *cobra.Command {
	crawlCmd := &cobra.Command{
		Use:   "crawl",
		Short: "Local crawl",
		Long:  "Invoke a local crawler for 1 or more packages",
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

	addCrawlerFlags(crawlCmd)

	return crawlCmd
}

func newLocalEnqueueCmd() *cobra.Command {
	localEnqueueCmd := &cobra.Command{
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

	localEnqueueCmd.Flags().IntVarP(&db.DefaultQueuePriority, "priority", "p", db.DefaultQueuePriority, "Priority level to use when adding items to the queue")
	localEnqueueCmd.Flags().StringVarP(&EnqueueReason, "reason", "r", EnqueueReason, "Reason to use for to-crawl entries")

	return localEnqueueCmd
}

func newQueueDeleteCmd() *cobra.Command {
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
	return queueDeleteCmd
}

func newPurgeTableCmd() *cobra.Command {
	purgeTableCmd := &cobra.Command{
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

				case db.TableMetadata, "metadata", "meta":
					if err := dbClient.Purge(db.TableMetadata); err != nil {
						return fmt.Errorf("delete all metadata entries: %s", err)
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
	return purgeTableCmd
}

func newGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
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
	return getCmd
}

func newLsCmd() *cobra.Command {
	lsCmd := &cobra.Command{
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
	return lsCmd
}

func newDeletePackageCmd() *cobra.Command {
	deletePackageCmd := &cobra.Command{
		Use:     "delete-package",
		Aliases: []string{"delete-packages"},
		Short:   "Delete a package from the database",
		Long:    "Delete a package from the database",
		Args:    cobra.MinimumNArgs(1),
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
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
