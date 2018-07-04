package domain

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/vcs"
)

func TestPackageKeyNormalization(t *testing.T) {
	pkg := &Package{
		Path: "gigawatt.io/foobar",
		Data: &PackageSnapshot{
			SubPackages: map[string]*SubPackage{
				"gigawatt.io/foobar/amazing/yep": &SubPackage{
					Name: "yep",
				},
				"gigawatt.io/foobar/else": &SubPackage{
					Name: "else",
				},
			},
		},
	}
	pkg.NormalizeSubPackageKeys()
	t.Logf("pkg=%# v", pkg.Data.SubPackages)
	if expected, actual := 2, len(pkg.Data.SubPackages); actual != expected {
		t.Errorf("Expected number of sub-packages=%v but actual=%v", expected, actual)
	}
	for _, normalized := range []string{"amazing/yep", "else"} {
		if _, ok := pkg.Data.SubPackages[normalized]; !ok {
			t.Errorf("Expected to find normalized sub-package key=%v in map but it was not there", normalized)
		}
	}
}

func TestPackageMergePending(t *testing.T) {
	var (
		importerPkgPath = "jaytaylor.com/archive.is"
		pkg             = NewPackage(newFakeRR("git@github.com:sirupsen/logrus", "github.com/sirupsen/logrus"))
		pendingRefs     = &PendingReferences{
			PackagePath: "github.com/sirupsen/logrus",
			ImportedBy: map[string]*PackageReferences{
				importerPkgPath: NewPackageReferences(NewPackageReference("github.com/sirupsen/logrus")),
			},
		}
	)
	// t.Logf("Before: %+v", *pkg)
	pkg.MergePending(pendingRefs)
	// t.Logf(" After: %+v", *pkg)
	if expected, actual := 1, len(pkg.ImportedBy); actual != expected {
		t.Errorf("Expeted len(pkg.ImportedBy)=%v but actual=%v", expected, actual)
	}

	// Then add logrus again with a newer timestamp and verify that .MarkSeen()
	// was invoked and the new timestamp matches expected value after merge.
	origRefs := pendingRefs.ImportedBy[importerPkgPath]
	newRefs := NewPackageReferences(NewPackageReference("github.com/sirupsen/logrus"))
	pendingRefs.ImportedBy[importerPkgPath] = newRefs

	pkg.MergePending(pendingRefs)

	// Verify FirstSeenAt and LastSeenAt values.
	if expected, actual := pkg.ImportedBy[importerPkgPath].Refs[0].LastSeenAt, newRefs.Refs[0].LastSeenAt; !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected pkg.LastSeenAt=%v but actual=%v", expected, actual)
	}
	if expected, actual := pkg.ImportedBy[importerPkgPath].Refs[0].FirstSeenAt, origRefs.Refs[0].FirstSeenAt; !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected pkg.FirstSeenAt=%v but actual=%v", expected, actual)
	}
}

func newFakeRR(repo string, root string) *vcs.RepoRoot {
	rr := &vcs.RepoRoot{
		Repo: repo,
		Root: root,
		VCS: &vcs.Cmd{
			Name: "git",
		},
	}
	return rr
}
