package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

var (
	HostsExtended bool
	MRUMaxItems   = 100
)

func newStatsCmd() *cobra.Command {
	statsCmd := &cobra.Command{
		Use:     "statistics",
		Aliases: []string{"stats", "stat", "st"},
		Short:   "Statistics information",
		Long:    "Statistics-related information",
	}

	statsCmd.AddCommand(
		newDBStatsCmd(),
		newHostsCmd(),
	)

	return statsCmd
}

func newDBStatsCmd() *cobra.Command {
	dbStatsCmd := &cobra.Command{
		Use:   "db",
		Short: "DB table-entry counts",
		Long:  "Displays packages and to-crawls table counts",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := db.WithClient(db.NewConfig(DBDriver, DBFile), func(dbClient db.Client) error {
				counts := map[string]int{}
				for _, table := range db.KVTables() {
					l, err := dbClient.Backend().Len(table)
					if err != nil {
						return fmt.Errorf("getting len(%v): %s", table, err)
					}
					counts[table] = l
				}
				for _, table := range db.QTables() {
					l, err := dbClient.Queue().Len(table, 0)
					if err != nil {
						return fmt.Errorf("getting len(%v): %s", table, err)
					}
					counts[table] = l
				}
				return emitJSON(counts)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}
	return dbStatsCmd
}

func newHostsCmd() *cobra.Command {
	hostsCmd := &cobra.Command{
		Use:   "hosts",
		Short: "Unique hosts",
		Long:  "Map of each unique host, and repo and package counts per host",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			var (
				dbCfg = db.NewConfig(DBDriver, DBFile)
				hosts interface{}
			)
			if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
				var err error
				if HostsExtended {
					hosts, err = uniqueHostsExtended(dbClient)
				} else {
					hosts, err = uniqueHosts(dbClient)
				}
				if err != nil {
					return err
				}
				return emitJSON(hosts)
			}); err != nil {
				log.Fatalf("main: %s", err)
			}
		},
	}

	hostsCmd.Flags().BoolVarP(&HostsExtended, "extended", "e", false, "Include repository and package counts per host")

	return hostsCmd
}

// uniqueHosts returns a slice of all unique hosts in the packages table.
func uniqueHosts(client db.Client) ([]string, error) {
	var (
		hostsMap = map[string]struct{}{}
		n        = 0
	)
	if err := client.EachRow(db.TablePackages, func(k []byte, _ []byte) {
		if n > 0 && n%10000 == 0 {
			log.WithField("n", n).Debug("Processed chunk")
		}
		n++
		if !bytes.Contains(k, []byte{'.'}) {
			return
		}
		h := string(bytes.Split(k, []byte{'/'})[0])
		if _, ok := hostsMap[h]; !ok {
			hostsMap[h] = struct{}{}
		}
	}); err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(hostsMap))
	for host, _ := range hostsMap {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts, nil
}

// uniqueHostsExtended returns a map of all unique hosts with repository and
// package counts per-host.
func uniqueHostsExtended(client db.Client) (map[string]map[string]int, error) {
	var (
		hosts = map[string]map[string]int{}
		n     = 0
	)
	if err := client.EachPackage(func(pkg *domain.Package) {
		if n > 0 && n%10000 == 0 {
			log.WithField("n", n).Debug("Processed chunk")
		}
		n++
		if !strings.Contains(pkg.Path, ".") {
			return
		}
		h := strings.Split(pkg.Path, "/")[0]
		if _, ok := hosts[h]; !ok {
			hosts[h] = map[string]int{}
		}

		if _, ok := hosts[h]["repos"]; !ok {
			hosts[h]["repos"] = 0
		}
		hosts[h]["repos"]++

		if _, ok := hosts[h]["packages"]; !ok {
			hosts[h]["packages"] = 0
		}
		hosts[h]["packages"] += len(pkg.Data.SubPackages)
	}); err != nil {
		return hosts, err
	}
	return hosts, nil
}

// Hosts() (HostStats, error)                                                                     // Map of hosts -> repo and package count per host.
// type HostStats map[string]map[string]int
