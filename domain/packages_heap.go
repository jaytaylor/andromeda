package domain

import (
	"container/heap"
)

// PackagesHeap facilitates building prioritized / sorted collections of
// packages.
type PackagesHeap struct {
	Packages []*Package
	LessFunc PackagesLessFunc
}

// PackagesLessFunc Package comparison function type.
type PackagesLessFunc func(a, b *Package) bool

func NewPackagesHeap(lessFn func(a, b *Package) bool) *PackagesHeap {
	ph := &PackagesHeap{
		Packages: []*Package{},
		LessFunc: lessFn,
	}
	heap.Init(ph)

	return ph
}

func (ph PackagesHeap) Len() int { return len(ph.Packages) }
func (ph PackagesHeap) Less(i, j int) bool {
	return ph.LessFunc(ph.Packages[i], ph.Packages[j])
}
func (ph PackagesHeap) Swap(i, j int) { ph.Packages[i], ph.Packages[j] = ph.Packages[j], ph.Packages[i] }

func (ph *PackagesHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	ph.Packages = append(ph.Packages, x.(*Package))
}
func (ph *PackagesHeap) PackagePush(pkg *Package) {
	heap.Push(ph, pkg)
}

func (ph *PackagesHeap) Pop() interface{} {
	old := ph.Packages
	n := len(old)
	x := old[n-1]
	ph.Packages = old[0 : n-1]
	return x
}
func (ph *PackagesHeap) PackagePop() *Package {
	if len(ph.Packages) == 0 {
		return nil
	}
	pkg := heap.Pop(ph).(*Package)
	return pkg
}

func (ph *PackagesHeap) Slice() []*Package {
	orig := make([]*Package, ph.Len())
	for i, p := range ph.Packages {
		orig[i] = p
	}

	s := make([]*Package, ph.Len())
	i := 0
	for ph.Len() > 0 {
		s[i] = ph.PackagePop()
		i++
	}

	ph.Packages = orig

	return s
}

var (
	// PackagesByFirstSeenAt comparison function which sorted by the FirstSeenAt
	// field.
	PackagesByFirstSeenAt = func(a, b *Package) bool {
		if a == nil || b == nil || a.FirstSeenAt == nil || b.FirstSeenAt == nil {
			return false
		}
		return a.FirstSeenAt.Before(*b.FirstSeenAt)
	}

	// PackagesByCommittedAt comparison function which sorted by the
	// CommittedAt field.
	PackagesByCommittedAt = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil || a.Data.CommittedAt == nil || b.Data.CommittedAt == nil {
			return false
		}
		return a.Data.CommittedAt.Before(*b.Data.CommittedAt)
		// return h[i].Data.CommittedAt.Before(*h[j].Data.CommittedAt)
	}

	// PackagesByLastSeenAt comparison function which sorted by the
	// Data.CreatedAt field.
	PackagesByLastSeenAt = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil || a.Data.CreatedAt == nil || b.Data.CreatedAt == nil {
			return false
		}
		return a.Data.CreatedAt.Before(*b.Data.CreatedAt)
	}

	// PackagesByBytesTotal comparison function which sorted by the
	// total checked out size.
	PackagesByBytesTotal = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil {
			return false
		}
		return a.Data.BytesTotal < b.Data.BytesTotal
	}

	// PackagesByBytesVCSDir comparison function which sorted by the
	// VCS directory size.
	PackagesByBytesVCSDir = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil {
			return false
		}
		return a.Data.BytesVCS < b.Data.BytesVCS
	}

	// PackagesByBytesData comparison function which sorted by the
	// actual repository content (code files, etc) size.
	PackagesByBytesData = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil {
			return false
		}
		return a.Data.BytesTotal-a.Data.BytesVCS < b.Data.BytesTotal-b.Data.BytesVCS
	}

	// PackagesByNumImports comparison function which sorted by the
	// number of imports a package uses.
	PackagesByNumImports = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil {
			return false
		}
		return len(a.Data.CombinedImports()) < len(b.Data.CombinedImports())
	}

	// PackagesByNumImportedBy comparison function which sorted by the
	// number of imports from other packages.
	PackagesByNumImportedBy = func(a, b *Package) bool {
		if a == nil || b == nil || a.Data == nil || b.Data == nil {
			return false
		}
		return len(a.ImportedBy) < len(b.ImportedBy)
	}

	// PackagesDesc returns a reversed version of the specified less function.
	PackagesDesc = func(lessFn PackagesLessFunc) PackagesLessFunc {
		fn := func(a, b *Package) bool {
			return !lessFn(a, b)
		}
		return fn
	}
)

/*type packagesHeap []*Package

func (h packagesHeap) Len() int { return len(h) }
func (h packagesHeap) Less(i, j int) bool {
	return h[i].Data.CommittedAt.Before(*h[j].Data.CommittedAt)
}
func (h packagesHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *packagesHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(*Package))
}

func (h *packagesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}*/
