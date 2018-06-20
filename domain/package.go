package domain

import (
	"reflect"
	"sort"
	"time"

	"golang.org/x/tools/go/vcs"

	"jaytaylor.com/andromeda/pkg/unique"
)

func NewPackage(path string) *Package {
	now := time.Now()

	pkg := &Package{
		Path:        path,
		FirstSeenAt: &now,
	}

	return pkg
}

func (pkg *Package) LatestCrawl() *PackageCrawl {
	l := len(pkg.History)
	if l == 0 {
		return nil
	}
	return pkg.History[l-1]
}

func (pkg *Package) MostlyEmpty() bool {
	return len(pkg.URL) == 0 && len(pkg.VCS) == 0 && pkg.Data == nil
}

func (pkg Package) RepoRoot() *vcs.RepoRoot {
	rr := &vcs.RepoRoot{
		Repo: pkg.URL,
		Root: pkg.Path,
		VCS: &vcs.Cmd{
			Name: pkg.VCS,
		},
	}
	return rr
}

func (pkg *Package) Merge(other *Package) *Package {
	if pkg == nil {
		pkg = &Package{}
	}

	if pkg.ID <= 0 && other.ID > 0 {
		pkg.ID = other.ID
	}
	if len(other.Name) > 0 {
		pkg.Name = other.Name
	}
	if len(other.Owner) > 0 {
		pkg.Owner = other.Owner
	}
	if len(other.Path) > 0 {
		pkg.Path = other.Path
	}
	if len(other.URL) > 0 {
		pkg.URL = other.URL
	}
	if len(other.VCS) > 0 {
		pkg.VCS = other.VCS
	}
	if other.Data != nil {
		pkg.Data = pkg.Data.Merge(other.Data)
	}
	if len(other.History) > 0 {
		pkg.History = append(pkg.History, other.History...)
	}
	if len(other.ImportedBy) > 0 {
		pkg.ImportedBy = unique.Strings(append(pkg.ImportedBy, other.ImportedBy...))
	}
	return pkg
}

func (snap *PackageSnapshot) AllImports() []string {
	impsMap := map[string]struct{}{}
	for _, imp := range append(snap.Imports, snap.TestImports...) {
		impsMap[imp] = struct{}{}
	}
	all := make([]string, 0, len(impsMap))
	for imp, _ := range impsMap {
		all = append(all, imp)
	}
	sort.Strings(all)
	return all
}

func (snap *PackageSnapshot) Merge(other *PackageSnapshot) *PackageSnapshot {
	if snap == nil {
		snap = &PackageSnapshot{}
	}

	if other.Repo != "" {
		snap.Repo = other.Repo
	}
	if !reflect.DeepEqual(snap.Imports, other.Imports) {
		snap.Imports = other.Imports
	}
	if !reflect.DeepEqual(snap.TestImports, other.TestImports) {
		snap.TestImports = other.TestImports
	}
	if !reflect.DeepEqual(snap.Deps, other.Deps) {
		snap.Deps = other.Deps
	}
	if other.Commits != int32(0) {
		snap.Commits = other.Commits
	}
	if other.Branches != int32(0) {
		snap.Branches = other.Branches
	}
	if other.Tags != int32(0) {
		snap.Tags = other.Tags
	}
	if other.Bytes != int64(0) {
		snap.Bytes = other.Bytes
	}
	if other.Stars != int32(0) {
		snap.Stars = other.Stars
	}
	if other.Readme != "" {
		snap.Readme = other.Readme
	}

	return snap
}

func (pc *PackageCrawl) AddMessage(msg string) {
	if pc.JobMessages == nil {
		pc.JobMessages = []string{}
	}
	pc.JobMessages = append(pc.JobMessages, msg)
}

func (pc PackageCrawl) Duration() time.Duration {
	if pc.JobStartedAt == nil || pc.JobFinishedAt == nil {
		return time.Duration(0)
	}
	d := pc.JobFinishedAt.Sub(*pc.JobStartedAt)
	return d
}
