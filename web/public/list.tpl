<!doctype html>
<html>
<head>
    <title>{{ len . }} Packages - Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">
</head>
<body>

{{ range $pkgPath, $pkg := . }}
<a href="/{{ $pkgPath }}">{{ $pkgPath }}</a>
<br>
{{ end }}

</body>
</html>
