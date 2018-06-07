package domain

import (
	"time"
)

func NewToCrawlEntry(pkgPath string, reason ...string) *ToCrawlEntry {
	if len(reason) == 0 {
		reason = []string{""}
	}

	now := time.Now()

	tce := &ToCrawlEntry{
		PackagePath: pkgPath,
		Reason:      reason[0],
		SubmittedAt: &now,
	}

	return tce
}
