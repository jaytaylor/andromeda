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
<div>URL: <a href="{{ .URL }}" ref="nofollow">{{ .URL }}</a></div>
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
<div>Data.Bytes: {{ .Data.Bytes }}</div>
<div>Data.Forks: {{ .Data.Forks }}</div>

{{ with $subPkg := index .Data.SubPackages "" }}
{{ if $subPkg.Imports }}
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
<hr>

{{ if .Data.SubPackages }}
{{ $pkg := . }}
<h3>{{ len .Data.SubPackages }} Nested Packages:</h3>
{{ range $subPkgPath, $subPkg := .SubPackagesPretty }}
<a href="/{{ $pkg.Path }}/{{ $subPkgPath }}">{{ $subPkgPath }}</a>
<br>
{{ end }}
{{ end }}

{{ if .ImportedBy }}
<h4>Imported By<h4>
<ul>
{{ range $imp := .ImportedBy }}
    <li><a href="/{{ $imp }}">{{ $imp }}</a></li>
{{ end }}
</ul>
<hr>
{{ end }}

<h4>Crawl History</h4>
{{ range $idx, $crawl := .History }}
<div>History[{{ $idx }}] Duration: {{ $crawl.Duration }}</div>
<div>History[{{ $idx }}] Started: {{ $crawl.JobStartedAt }}</div>
<div>History[{{ $idx }}] Finished: {{ $crawl.JobFinishedAt }}</div>
<div>History[{{ $idx }}] Succeeded: {{ $crawl.JobSucceeded }}</div>
<div>History[{{ $idx }}] Messages: {{ $crawl.JobMessages }}</div>
<hr>
{{ end }}

</body>
</html>
