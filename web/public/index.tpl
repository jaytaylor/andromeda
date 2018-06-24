<!doctype html>
<html>
<head>
    <title>Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">
</head>
<body>
<h1>Andromeda</h1>
<h2>Search the entire visible Golang Universe</h2>
<br>
{{ with $latest := .Config.Master.Latest }}
{{ if $latest }}
<div>
    <h3>Latest Crawled Packages</h3>
    <ul style="list-style-type: none">
{{ range $pkg := $latest }}
        <li>{{ $pkg.Data.CreatedAt }} <a href="/{{ $pkg.Path }}">{{ $pkg.Path }}</a></li>
{{ end }}
    </ul>
</div>
{{ end }}
{{ end }}
Number of packages in index: {{ .DB.PackagesLen }}
<br>
Crawl queue length: {{ .DB.ToCrawlsLen }}
<br>
Active workers: {{ index .Config.Master.Stats "remotes" }}
<br>
Crawls since program started: {{ index .Config.Master.Stats "crawls" }}
</body>
</html>
