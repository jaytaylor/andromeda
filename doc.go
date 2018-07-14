package andromeda

// Package andromeda is a system for building a graph over the entire visible
// universe of golang.
//
// Overview
//
// The system is comprised of the following component stages:
//
// 1. Seed data set
//
// The initial set of packages selected to be fed into Andromeda's to-crawl
// queue is important because this decision sets the trajectory for what
// packages will be crawled ingested into the graph.
//
// Godoc.org
//
// If the goal is to build as complete of a picture of the public go landscape
// as possible, starting with the packages listing from godoc.org is a good
// starting place.
//
// Regarding efficiency: It's worth noting that a substantial 5-8x reduction in
// redundant crawls is achievable by merging out redundant package listings.
//
// One technique is to exploit knowledge of git repository root structures for
// popular git hosting provider services (github, bitbucket, etc.) to eliminate
// duplicate entries.
//
// For example, given the following set of packages:
//
//     github.com/gohugoio/hugo/hugolib/pagemeta
//     github.com/gohugoio/hugo/hugolib/paths
//     github.com/gohugoio/hugo/tpl
//     github.com/gohugoio/hugo/tpl/crypto
//     github.com/gohugoio/hugo/tpl/math
//
// Since we know all github repositories have the URL structure:
//
//     github.com/[USER]/[REPOSITORY]
//
// The list of input packages can be sorted, trimmed everything after
// user/repository portion, then remove duplicate lines.  Leaving only the
// necessary repository root:
//
//     github.com/gohugoio/hugo
//
// 2. Load seed data set into Andromeda
//
// 3. Begin crawling
//
// * Run Andromeda web server
// * Run Andromeda crawler(s)
//
