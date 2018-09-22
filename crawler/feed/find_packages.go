package feed

import (
	"regexp"
	"strings"

	"mvdan.cc/xurls"

	"jaytaylor.com/andromeda/pkg/unique"
)

var (
	validPkgExpr  = regexp.MustCompile(`^[^.]+.[^\/]+(?:\/[a-zA-Z0-9_.-]*)*$`)
	cleanExpr     = regexp.MustCompile(`[^a-zA-Z0-9:\/_.-]+`)
	bannedDomains = []string{
		"ycombinator.com",
		"reddit.com",
		"youtube.com",
		"youtu.be",
		"ytimg.com",
		"imgur.com",
		"bit.ly",
		// "nytimes.com",
		// "wired.com",
		// "wsj.com",
		// "bloomberg.com",
		// "slate.com",
		// "wikipedia.org",
		// "stackoverflow.com",
		// "stackexchange.com",
		// "medium.com",
		// "qz.com",
		// "reuters.com",
		// "arxiv.org",
		// "archive.is",
		// "archive.org",
		// "theverge.com",
		// "latimes.com",
		// "newyorder.com",
		// "boingboing.net",
		// "arstechnica.com",
		// "techcrunch.com",
		// "washingtonpost.com",
		// "googleblog.com",
		// "theguardian.com",
		// "nature.com",
		// "groups.google.com",
		// "fda.gov",
		// "vice.com",
		// "yahoo.com",
		// "wordpress.com",
		// "economist.com",
		// "acm.org",
		// "smithsonianmag.com",
		// "politico.com",
		// "scientificamerican.com",
		// "fortune.com",
		// "cnn.com",
		// "npr.org",
		// "cnbc.com",
		// "gizmodo.com",
		// "quora.com",
		// "engadget.com",
		// "dice.com",
		// "theatlantic.com",
		// "sciencemag.org",
		// "wikileaks.org",
		// "theintercept.com",
		// "forbes.com",
		// "fastcompany.com",
		// "sfgate.com",
		// "rollingstone.com",
		// "bbc.com",
		// "plus.google.com",
		// ".blogspot.com",
		// "torrentfreak.com",
		// "mercurynews.com",
	}
)

// FindPackages identifies all possible package paths in a block of text.
func FindPackages(text string) []string {
	text = cleanExpr.ReplaceAllString(text, ` $1 `)
	matches := xurls.Relaxed().FindAllString(text, -1)
	accepted := []string{}
	for i := 0; i < len(matches); i++ {
		if strings.Contains(matches[i], "://") {
			matches[i] = strings.Split(matches[i], "://")[1]
		}
		matches[i] = strings.TrimRight(matches[i], "/")
		if validPkgExpr.MatchString(matches[i]) && !isBannedDomain(matches[i]) {
			accepted = append(accepted, matches[i])
		}
	}
	u := unique.Strings(accepted)
	return u
}

func isBannedDomain(u string) bool {
	d := strings.SplitN(u, "/", 2)[0]
	for _, b := range bannedDomains {
		if strings.HasSuffix(d, b) {
			return true
		}
	}
	return false
}
