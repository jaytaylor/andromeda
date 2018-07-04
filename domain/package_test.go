package domain

import (
	"testing"
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
