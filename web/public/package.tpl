{{ $pkg := . }}
<!doctype html>
<html>
<head>
    <title>{{ .Path }} - Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">
</head>
<body>

<h1>{{ .RepoName }}</h1>

<div>
<a href="/">/</a>
{{ range $parentPath := .ParentPaths }}
<a href="/{{ $parentPath.Path }}">{{ $parentPath.Name }}</a>
{{ end }}
</div>
<hr>

<div>ID: {{ .ID }}</div>
<!--<h3>Package Root: {{ .Path }}</h3>-->
<div>Repository Root: <a href="{{ .URL }}" ref="nofollow">{{ .URL }}</a></div>
{{ with $webURL := $pkg.WebURL }}
{{ if ne $webURL $pkg.URL }}
<div>Source code web link: <a href="{{ $webURL }}">{{ $webURL }}</a></div>
{{ end }}
{{ end }}

<!-- <div>Name: {{ .Name }}</div> -->
<!-- <div>Owner: {{ .Owner }}</div> -->
<div>VCS: {{ .VCS }}</div>
<div>First seen: {{ .FirstSeenAt }}</div>
<div><a href="/v1/{{ .Path }}">JSON</a></div>
<div><a href="https://godoc.org/{{ .Path }}">Documentation</a></div>
<hr>

<h3>Package Information</h3>
{{ if .Data }}
<div>Data.Repo: {{ .Data.Repo }}</div>
<div>Data.Commits: {{ .Data.Commits }}</div>
<div>Data.Branches: {{ .Data.Branches }}</div>
<div>Data.Tags: {{ .Data.Tags }}</div>
<div>Data.BytesTotal: {{ .Data.PrettyBytesTotal }}</div>
<div>Data.Forks: {{ .Data.Forks }}</div>

{{ if .ImportedBy }}
<hr>
<h4>Imported By {{ len .ImportedBy }} package{{ len .ImportedBy | plural "" "s" }}<h4>
<ul>
{{ range $imp, $refs := .ImportedBy }}
    <li><a href="/{{ $imp }}">{{ $imp }} ({{ len $refs.Refs }} reference{{ len $refs.Refs | plural "" "s" }})</a></li>
{{ end }}
</ul>
{{ end }}

{{ with $allImports := $pkg.Data.AllImports }}
{{ if gt (len $allImports) 0 }}
<hr>
<div>
<h4>All Imports</h4>
<ul style="list-style-type: none">
{{ range $imp := $allImports }}
    <li><a href="/{{ $imp }}">{{ $imp }}</a></li>
{{ end }}
</ul>
</div>
{{ end }}
{{ end }}

{{ with $allTestImports := $pkg.Data.AllTestImports }}
{{ if gt (len $allTestImports) 0 }}
<hr>
<div>
<h4>All Test Imports</h4>
<ul style="list-style-type: none">
{{ range $imp := $allTestImports }}
    <li><a href="/{{ $imp }}">{{ $imp }}</a></li>
{{ end }}
</ul>
</div>
{{ end }}
{{ end }}



{{ with $subPkg := index .Data.SubPackages "" }}
{{ if $subPkg.Imports }}
<hr>
<div>
<h4>Imports</h4>
<ul style="list-style-type: none">
{{ range $imp := $subPkg.Imports }}
    <li><a href="/{{ $imp }}">{{ $imp }}</a></li>
{{ end }}
</ul>
</div>
{{ end }}

{{ if and $subPkg $subPkg.Readme }}
<hr>
<div>Data.Readme: <pre>{{ $subPkg.Readme }}</pre></div>
{{ end }}
{{ end }}
{{ end }}

{{ if .Data.SubPackages }}
<hr>
<h3>{{ len .Data.SubPackages }} Nested Package{{ len .Data.SubPackages | plural "" "s" }}:</h3>
{{ range $subPkgPath, $subPkg := .SubPackagesPretty }}
<a href="/{{ $subPkgPath }}">{{ $subPkgPath }}</a>
<br>
{{ end }}
{{ end }}

<hr>
<h4>Crawl History</h4>
{{ range $idx, $crawl := .History }}
<div>History[{{ $idx }}] Duration: {{ $crawl.Duration }}</div>
<div>History[{{ $idx }}] Started: {{ $crawl.JobStartedAt }}</div>
<div>History[{{ $idx }}] Finished: {{ $crawl.JobFinishedAt }}</div>
<div>History[{{ $idx }}] Succeeded: {{ $crawl.JobSucceeded }}</div>
<div>History[{{ $idx }}] Messages: {{ $crawl.JobMessages }}</div>
{{ end }}
<hr>

</body>
</html>
