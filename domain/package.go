package domain

import (
	"reflect"
	"time"
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

func (pc *PackageCrawl) AddMessage(msg string) {
	if pc.JobMessages == nil {
		pc.JobMessages = []string{}
	}
	pc.JobMessages = append(pc.JobMessages, msg)
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
