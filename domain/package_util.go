package domain

import (
	"regexp"
	"strings"
)

var gitURLExpr = regexp.MustCompile(`^(?:[hH][tT][tT][pP][sS]?:\/\/|.+@(.+\..+:))?(.+?)(?:\.git)?$`)

// PackagePathFromURL supports extracting a go package name from a git clone URL.
// Returns the cleaned package path name and a boolean indicating whether or not
// the request should be redirected.
func PackagePathFromURL(name string) (string, bool) {
	name = strings.Trim(name, "/")
	pkgPath := name
	pkgPath = gitURLExpr.ReplaceAllString(pkgPath, "$1$2")
	redirect := pkgPath != name
	pkgPath = strings.Replace(pkgPath, ":", "/", 1)
	return pkgPath, redirect
}
