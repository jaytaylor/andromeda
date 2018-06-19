package crawler

import (
	"golang.org/x/tools/go/vcs"
)

// PackageResolver is the interface for structs who know how to resolve a
// pacakge to a *vcs.RepoRoot, one way or another.
type PackageResolver interface {
	Resolve(pkgPath string) (*vcs.RepoRoot, error)
}
