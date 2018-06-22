package domain

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"golang.org/x/tools/go/vcs"
)

// NewPackage turns a *vcs.RepoRoot into a new *Package.  If now is omitted or
// nil, the current time will be used.
func NewPackage(rr *vcs.RepoRoot, now ...*time.Time) *Package {
	if len(now) == 0 || now[0] == nil {
		ts := time.Now()
		now = []*time.Time{&ts}
	}
	pkg := &Package{
		FirstSeenAt: now[0],
		Path:        rr.Root,
		Name:        "", // TODO: package name(s)???
		URL:         rr.Repo,
		VCS:         rr.VCS.Name,
		Data:        &PackageSnapshot{},
		ImportedBy:  map[string]*PackageReferences{},
		History:     []*PackageCrawl{},
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

// RepoRoot constructs a "fake" repository root, which may be adequate in some
// cases.
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

// func NewSubPackage(path string, name string) {
// 	subPkg := &SubPackage{
// 		Path: path,
// 		Name: name,
// 	}
// 	return subPkg
// }

func (pkg *Package) Merge(other *Package) *Package {
	// TODO: When merging packages, it's going to be important to compare the the
	// imports this time to the previous imports, and generate some diffs to
	// update the newly referenced (or no longer referenced) packages.

	if pkg == nil {
		pkg = &Package{}
	}

	if pkg.ID == 0 && other.ID > 0 {
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
		if pkg.Data != nil {
			pkg.Data = pkg.Data.Merge(other.Data)
		} else {
			pkg.Data = other.Data
		}
	}
	if len(other.History) > 0 {
		pkg.History = append(pkg.History, other.History...)
	}
	if len(other.ImportedBy) > 0 {
		// pkg.ImportedBy = unique.Strings(append(pkg.ImportedBy, other.ImportedBy...))
		for subPkgPath, wrapper := range other.ImportedBy {
			for _, ref := range wrapper.Refs {
				pkg.UpdateImportedBy(subPkgPath, ref)
			}
		}
	}
	return pkg
}

// UpdateImportedBy behaves differently based on whether pkgRef.Active is true or
// false.
//
// When pkgRef.Active is true it adds or updates the timestamp on an importer.
//
// When pkgRef.Active is false it will be marked as inactive if the record is
// found, otherwise it'll be ignored.
func (pkg *Package) UpdateImportedBy(subPkgPath string, pkgRef *PackageReference) {
	subPkgPath = SubPackagePathNormalize(pkg.Path, subPkgPath)

	if wrapper, ok := pkg.ImportedBy[subPkgPath]; ok {
		for _, ref := range wrapper.Refs {
			if pkgRef.Path == ref.Path {
				if pkgRef.Active {
					ref.LastSeenAt = pkgRef.LastSeenAt
				} else {
					ref.Active = false
				}
				return
			}
		}
		// TODO: Sort alphabetically or by a ranking metric.
		pkg.ImportedBy[subPkgPath].Refs = append(pkg.ImportedBy[subPkgPath].Refs, pkgRef)
	} else {
		pkg.ImportedBy[subPkgPath] = &PackageReferences{Refs: []*PackageReference{pkgRef}}
	}
}

func NewPackageReference(path string) *PackageReference {
	now := time.Now()
	pr := &PackageReference{
		Path:        path,
		Active:      true,
		FirstSeenAt: &now,
		LastSeenAt:  &now,
	}
	return pr
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
	// if snap == nil {
	// 	snap = &PackageSnapshot{}
	// }

	if snap.CreatedAt == nil && other.CreatedAt != nil {
		snap.CreatedAt = other.CreatedAt
	}
	if other.Repo != "" {
		snap.Repo = other.Repo
	}
	if !reflect.DeepEqual(snap.SubPackages, other.SubPackages) {
		snap.SubPackages = other.SubPackages
	}
	if !reflect.DeepEqual(snap.Imports, other.Imports) {
		snap.Imports = other.Imports
	}
	if !reflect.DeepEqual(snap.TestImports, other.TestImports) {
		snap.TestImports = other.TestImports
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
	if other.Bytes != uint64(0) {
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

func (snap PackageSnapshot) PrettyBytes() string {
	pretty := humanize.Bytes(snap.Bytes)
	return pretty
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

func SubPackagePathNormalize(pkgPath string, subPkgPath string) string {
	if subPkgPath == pkgPath {
		// NB: An empty string signifies the root package path.
		return ""
	}
	normalized := strings.Replace(subPkgPath, pkgPath+"/", "", 1)
	return normalized
}

func SubPackagePathDenormalize(pkgPath string, subPkgPath string) string {
	if subPkgPath == "" {
		return pkgPath
	}
	denormalized := fmt.Sprintf("%v/%v", pkgPath, subPkgPath)
	return denormalized
}
