package main

import (
	"bytes"
	"container/heap"
	"fmt"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
)

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.AddCommand(dbStatsCmd)

	statsCmd.AddCommand(hostsCmd)

	hostsCmd.Flags().BoolVarP(&HostsExtended, "extended", "e", false, "Include repository and package counts per host")

	statsCmd.AddCommand(mruStatsCmd)

	mruStatsCmd.Flags().IntVarP(&MRUMaxItems, "num", "n", MRUMaxItems, "Number of most-recent committed packages to return")
}

var (
	HostsExtended bool
	MRUMaxItems   = 100
)

var statsCmd = &cobra.Command{
	Use:     "statistics",
	Aliases: []string{"stats", "stat", "st"},
	Short:   "Statistics information",
	Long:    "Statistics-related information",
}

var dbStatsCmd = &cobra.Command{
	Use:   "db",
	Short: "DB table-entry counts",
	Long:  "Displays packages and to-crawls table counts",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			pl, err := dbClient.PackagesLen()
			if err != nil {
				return fmt.Errorf("getting packages count: %s", err)
			}
			log.WithField("packages", pl).Debug("count")

			tcl, err := dbClient.ToCrawlsLen()
			if err != nil {
				return fmt.Errorf("getting to-crawls count: %s", err)
			}
			log.WithField("to-crawls", tcl).Debug("count")
			counts := map[string]int{
				"packages":  pl,
				"to-crawls": tcl,
			}
			return emitJSON(counts)
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Unique hosts",
	Long:  "Map of each unique host, and repo and package counts per host",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			dbCfg = db.NewBoltConfig(DBFile)
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

var mruStatsCmd = &cobra.Command{
	Use:   "mru",
	Short: "Get top N most recent packages by commit date",
	Long:  "Get top N most recent packages by commit date",
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if MRUMaxItems <= 0 {
			log.Fatal("Invalid value for --max/-m, must be an integer greater than 0")
		}
		if err := db.WithClient(db.NewBoltConfig(DBFile), func(dbClient db.Client) error {
			h := &MRUPackagesHeap{}
			heap.Init(h)
			if err := dbClient.EachPackage(func(pkg *domain.Package) {
				if pkg.Data.CommittedAt == nil {
					return
				}
				if h.Len() < MRUMaxItems || pkg.Data.CommittedAt.After(*(*h)[0].Data.CommittedAt) {
					heap.Push(h, pkg)
				}
				if h.Len() > MRUMaxItems {
					heap.Pop(h)
				}
			}); err != nil {
				return err
			}
			return emitJSON(h)
		}); err != nil {
			log.Fatalf("main: %s", err)
		}
	},
}

type MRUPackagesHeap []*domain.Package

func (h MRUPackagesHeap) Len() int { return len(h) }
func (h MRUPackagesHeap) Less(i, j int) bool {
	return h[i].Data.CommittedAt.Before(*h[j].Data.CommittedAt)
}
func (h MRUPackagesHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *MRUPackagesHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*domain.Package))
}

func (h *MRUPackagesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
