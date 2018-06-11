package domain

import (
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
