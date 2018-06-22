<!doctype html>
<html>
<head>
    <title>{{ .Path }} - Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">
</head>
<body>

<div>ID: {{ .ID }}</div>
<h3>Package Root: {{ .Path }}</h3>
<div>URL: <a href="{{ .URL }}" ref="nofollow">{{ .URL }}</a></div>
<!-- <div>Name: {{ .Name }}</div> -->
<!-- <div>Owner: {{ .Owner }}</div> -->
<div>VCS: {{ .VCS }}</div>
<div>First seen: {{ .FirstSeenAt }}</div>
<hr>

{{ if .Data }}
<div>Data.Repo: {{ .Data.Repo }}</div>
<div>Data.Commits: {{ .Data.Commits }}</div>
<div>Data.Branches: {{ .Data.Branches }}</div>
<div>Data.Tags: {{ .Data.Tags }}</div>
<div>Data.Bytes: {{ .Data.Bytes }}</div>
<div>Data.Forks: {{ .Data.Forks }}</div>
{{ if (index .Data.SubPackages "") }}
<div>Data.Readme: <pre>{{ (index .Data.SubPackages "").Readme }}</pre></div>
{{ end }}
<hr>
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
