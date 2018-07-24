package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestPackagesHeap(t *testing.T) {
	lessFn := func(a, b *Package) bool {
		return len(a.Path) > len(b.Path)
	}
	h := NewPackagesHeap(lessFn)
	paths := []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff", "ggggggg"}
	for _, p := range paths {
		h.PackagePush(&Package{Path: p})
	}
	{
		expected := []*Package{
			&Package{Path: "ggggggg"},
			&Package{Path: "ffffff"},
			&Package{Path: "eeeee"},
			&Package{Path: "dddd"},
			&Package{Path: "ccc"},
			&Package{Path: "bb"},
			&Package{Path: "a"},
		}
		if actual := h.Slice(); !reflect.DeepEqual(actual, expected) {
			t.Fatalf("Expected Slice()=%+v but actual=%v", expected, actual)
		}
	}
}

func TestPackagesHeapFuncs(t *testing.T) {
	h := NewPackagesHeap(PackagesDesc(PackagesByFirstSeenAt))
	paths := []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff", "ggggggg"}
	expected := make([]*Package, len(paths))
	for i, p := range paths {
		now := time.Now()
		pkg := &Package{
			Path:        p,
			FirstSeenAt: &now,
		}
		h.PackagePush(pkg)
		expected[len(paths)-i-1] = pkg
	}
	if actual := h.Slice(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Expected Slice()=%+v but actual=%v", expected, actual)
	}
}
