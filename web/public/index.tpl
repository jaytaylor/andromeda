<!doctype html>
<html>
<head>
    <title>Andromeda</title>
    <meta charset="utf-8">
    <meta name="author" content="Jay Taylor">
    <meta name="description" content="Andromeda is a graph of the entire known visible go universe">

    <script type="text/javascript">
        window.onload = function () {
            var maxLogLines = 14;
            var conn;
            var status = document.getElementById('status');
            var msg = document.getElementById('msg');
            var log = document.getElementById('log');
            function appendLog(item) {
                var doScroll = log.scrollTop > log.scrollHeight - log.clientHeight - 1;
                log.appendChild(item);
                if (doScroll) {
                    log.scrollTop = log.scrollHeight - log.clientHeight;
                }
                if (log.childElementCount > maxLogLines) {
                    log.removeChild(log.children[0]);
                }
            }
            /*document.getElementById('form').onsubmit = function () {
                if (!conn) {
                    return false;
                }
                if (!msg.value) {
                    return false;
                }
                conn.send(msg.value);
                msg.value = '';
                return false;
            };*/
            if (window['WebSocket']) {
                var proto = 'https:' == document.location.protocol ? 'wss' : 'ws';
                var wsConnect = function() {
                    status.innerHTML = '<b>Connecting..</b>';

                    conn = new WebSocket(proto + '://' + document.location.host + '/ws');
                    conn.onopen = function(evt) {
                        console.log(evt);
                        status.innerHTML = '<b>Connected</b>';
                    };
                    /*conn.onerror = function(evt) {
                        console.log(evt);
                        var item = document.createElement('div');
                        item.innerHTML = '<b>ERROR: ' + evt + '</b>';
                        appendLog(item);
                    };*/
                    conn.onclose = function(evt) {
                        status.innerHTML = '<b>Connection closed.</b>';
                        setTimeout(wsConnect, 10000);
                    };
                    conn.onmessage = function(evt) {
                        var messages = evt.data.split('\n');
                        for (var i = 0; i < messages.length; i++) {
                            var item = document.createElement('div');
                            item.innerText = messages[i];
                            appendLog(item);
                        }
                    };
                };
                wsConnect();
            } else {
                var item = document.createElement('div');
                item.innerHTML = '<b>Your browser does not support WebSockets.</b>';
                appendLog(item);
            }
        };
    </script>
    <style type="text/css">
        /*html {
            overflow: hidden;
        }*/
        #stream-container {
            padding: 0;
            margin: 0;
            width: 800px;
            height: 300px;
            min-height: 300px;
            max-height: 300px;
            background: gray;
        }
        #log {
            background: white;
            margin: 0;
            padding: 0; /*0.5em 0.5em 0.5em 0.5em;*/
            width: 800px;
            height: 250px;
            min-height: 250px;
            max-height: 250px;
            overflow: auto;
        }
        #form {
            padding: 0 0.5em 0 0.5em;
            margin: 0;
            /*position: absolute;*/
            /*bottom: 1em;*/
            /*left: 0px;*/
            width: 300px;
            /*overflow: hidden;*/
        }
    </style>

    <style type="text/css">
        html {
            /*font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;*/
            font-family: sans-serif;
            -webkit-text-size-adjust: 100%;
            -ms-text-size-adjust: 100%;
            color: #333333;
        }
        a {
            color: #375EAB;
            text-decoration: none;
        }
        a:active {
            text-decoration: underline;
        }
        ul {
            list-style-type: none;
        }
        .top-nav {
            background-color: #E0EBF5;
            border: 1px solid #D1E1F0;
        }
        .top-package {
            background-color: #EEEEEE;
        }
    </style>

</head>
<body>
<div class="top-nav">
    <ul>
        <li><h1><a href="/">Andromeda</a></h1></li>
        <li><a href="/about">About</a></li>
    </ul>
</div>
<h2>Search the entire visible Golang Universe</h2>
<br>

<div id="stream-container">
    <h3>Crawl Stream</h3>
    <div> Connection status: <span id="status"></span></div>
    <div id="log"></div>
</div>
<br>

{{- with $ctx := . }}

    {{- with $latest := .Config.Master.Latest -}}
        {{- if $latest -}}
<div>
    <h3>Latest Crawled Packages</h3>
    <ul style="list-style-type: none">
            {{ range $pkg := $latest -}}
        <li>{{ index $pkg "CreatedAt" }} <a href="/{{ index $pkg "Path" }}">{{ index $pkg "Path" }}</a></li>
            {{- end }}
    </ul>
</div>
        {{- end -}}
    {{- end }}

    {{- with $db := .DB }}
Number of indexed repositories: {{ $db.PackagesLen }}
<br>

Crawl queue: {{ $db.ToCrawlsLen }}
<br>
        {{- range $i, $_ := N 11 }}
            {{- if $i -}}
                {{- with $len := $db.Queue.Len "to-crawl" $i -}}
                    {{- if $len -}}
&nbsp;&nbsp;&nbsp;&nbsp;priority={{ $i }}: {{ $len }}
<br>
                    {{- end -}}
                {{- end -}}
            {{- end -}}
        {{- end }}
<br>

Results queue:
        {{ with $unprocessed := $db.Queue.Len "crawl-result" 0 -}}
{{ $unprocessed }}
        {{- else -}}
all caught up :)
        {{- end }}
<br>

    {{- end }}

Active workers: {{ index $ctx.Config.Master.Stats "remotes" }}
<br>
Crawls since program started: {{ index $ctx.Config.Master.Stats "crawls" }}
<br>
{{- end }}

</body>
</html>
