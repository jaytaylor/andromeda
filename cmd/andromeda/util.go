package main

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

// newRepoRootCmd TODO: move this to util sub-command.
func newRepoRootCmd() *cobra.Command {
	repoRootCmd := &cobra.Command{
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
	return repoRootCmd
}

// newRebuildDBCmd TODO: move this to util sub-command.
func newRebuildDBCmd() *cobra.Command {
	rebuildDBCmd := &cobra.Command{
		Use:   "rebuild-db",
		Short: "Rebuilds the database",
		Long:  "Rebuilds the entire database",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			dbCfg := db.NewConfig(DBDriver, DBFile)
			if err := db.WithClient(dbCfg, func(dbClient *db.Client) error {
				newCfg := db.NewBoltConfig(RebuildDBFile)
				newBe := db.NewBoltBackend(newCfg)
				return dbClient.RebuildTo(newBe)
			}); err != nil {
				log.Fatal(err)
			}
		},
	}

	rebuildDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	return rebuildDBCmd
}

// newRebuildAndCleanupDBCmd TODO: move this to util sub-command.
func newRebuildAndCleanupDBCmd() *cobra.Command {
	rebuildAndCleanupDBCmd := &cobra.Command{
		Use:   "rebuild-cleanup-db",
		Short: "Rebuilds the database and cleans up records",
		Long:  "Rebuilds the entire database and applies cleanup filters to the records along the way",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			var (
				dbCfg = db.NewConfig(DBDriver, DBFile)
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

			if err := db.WithClient(dbCfg, func(dbClient *db.Client) error {
				newCfg := db.NewBoltConfig(RebuildDBFile)
				newBe := db.NewBoltBackend(newCfg)
				return dbClient.RebuildTo(newBe, filterFn)
			}); err != nil {
				log.Fatal(err)
			}
		},
	}

	rebuildAndCleanupDBCmd.Flags().StringVarP(&RebuildDBFile, "target", "t", RebuildDBFile, "Target destination filename")

	return rebuildAndCleanupDBCmd
}
