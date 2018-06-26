<!doctype html>
<html>
<head>
    <title>{{ .Path }} - Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">
</head>
<body>

<h1>{{ .Path }}</h1>

<div>
<a href="/">/</a>
{{ range $parentPath := .Pkg.ParentPaths .Path }}
<a href="/{{ $parentPath.Path }}">{{ $parentPath.Name }}</a>
{{ end }}
</div>
<hr>

<h3>Parent Package</h3>
<div>ID: {{ .Pkg.ID }}</div>
<div>See: <a href="/{{ .Pkg.Path }}">{{ .Pkg.Path }}</a></div>
<div><a href="https://godoc.org/{{ .Path }}">Documentation</a></div>
<hr>

<h3>Package Information</h3>
{{ if .Pkg.Data }}
<div>Data.Repo: {{ .Pkg.Data.Repo }}</div>
<div>Data.Commits: {{ .Pkg.Data.Commits }}</div>
<div>Data.Branches: {{ .Pkg.Data.Branches }}</div>
<div>Data.Tags: {{ .Pkg.Data.Tags }}</div>
<div>Data.Bytes: {{ .Pkg.Data.Bytes }}</div>
<div>Data.Forks: {{ .Pkg.Data.Forks }}</div>

<div>
<h4>Imports</h4>
<ul style="list-style-type: none">
{{ range $imp := .Sub.Imports }}
    <li><a href="/{{ $imp }}">{{ $imp }}</a></li>
{{ end }}
</ul>
</div>
{{ end }}

{{ if .Sub.Readme }}
<hr>
<div>Data.Readme: <pre>{{ .Sub.Readme }}</pre></div>
{{ end }}
<hr>


{{ if .Sub.ImportedBy }}
<h4>Imported By<h4>
<ul>
{{ range $imp := .Sub.ImportedBy }}
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
