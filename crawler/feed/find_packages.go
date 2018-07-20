package feed

import (
	"regexp"
	"strings"

	"mvdan.cc/xurls"

	"jaytaylor.com/andromeda/pkg/unique"
)

var (
	validPkgExpr = regexp.MustCompile(`^[^.]+.[^\/]+(?:\/[a-zA-Z0-9_.-]*)*$`)
	cleanExpr    = regexp.MustCompile(`[^a-zA-Z0-9:\/_.-]+`)
)

// findPackages identifies all possible package paths in a block of text.
func findPackages(text string) []string {
	text = cleanExpr.ReplaceAllString(text, ` $1 `)
	matches := xurls.Relaxed().FindAllString(text, -1)
	accepted := []string{}
	for i := 0; i < len(matches); i++ {
		if strings.Contains(matches[i], "://") {
			matches[i] = strings.Split(matches[i], "://")[1]
		}
		matches[i] = strings.TrimRight(matches[i], "/")
		if validPkgExpr.MatchString(matches[i]) {
			accepted = append(accepted, matches[i])
		}
	}
	u := unique.Strings(accepted)
	return u
}
