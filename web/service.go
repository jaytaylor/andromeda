package web

// go : generate bash -c "( test ${SKIP_JS}x = 1x || ( echo 'Compiling frontend assets' && cd .. && ( test ! -f package.json || ( npm install && ( command -v gulp 1>/dev/null 2>&1 && gulp build || node_modules/.bin/gulp build ) ) ) ) ) && echo 'Generating public package from go-bindata Asset: ../public/*' && mkdir -p ../public && cd ../public && go-bindata -pkg='public' -o public.go -ignore=public\\.go `find . -type d` && gofmt -w public.go || (echo 'oops.. there was a problem, you may need to install go-bindata or fix frontend stuff, see README.md' && exit 1)"

import (
	"fmt"
	"html/template"
	"net/http"
	//"os"
	//"strings"
	//"time"

	//"gigawatt.io/errorlib"
	"gigawatt.io/web"
	"gigawatt.io/web/route"
	"github.com/nbio/hitch"
	log "github.com/sirupsen/logrus"

	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/web/public"
)

// TODO: Maybe rename WebService to Service, or relocate this pkg?

//var AsyncRequestHandlerTimeout time.Duration = 5 * time.Second

type Config struct {
	Addr    string
	DevMode bool
}

type WebService struct {
	*web.WebServer

	Config *Config
	DB     db.DBClient
}

func New(db db.DBClient, cfg *Config) *WebService {
	service := &WebService{
		Config: cfg,
		DB:     db,
	}
	options := web.WebServerOptions{
		Addr:    service.Config.Addr,
		Handler: service.activateRoutes().Handler(),
	}
	service.WebServer = web.NewWebServer(options)
	return service
}

func (service *WebService) activateRoutes() *hitch.Hitch {
	assetProvider := service.staticFilesAssetProvider()

	routes := []route.RouteMiddlewareBundle{
		route.RouteMiddlewareBundle{
			Middlewares: []func(http.Handler) http.Handler{
				service.LoggerMiddleware,
				web.StaticFilesMiddleware(assetProvider),
			},
			RouteData: []route.RouteDatum{
				{"get", "/", service.tplAssetEndpoint("index.tpl")},
			},
		},
	}
	h := route.Activate(routes)
	return h
}

func (service *WebService) tplAssetEndpoint(asset string) func(w http.ResponseWriter, req *http.Request) {
	// TODO: consider moving this outside of closure?  any benefit?
	log.Infof("NAMES: %s", public.AssetNames())
	content, err := service.staticFilesAssetProvider()(asset)
	if err != nil {
		panic(fmt.Errorf("problem with asset %q: %v", asset, err))
	}
	tpl := template.Must(template.New(asset).Parse(string(content)))

	return func(w http.ResponseWriter, req *http.Request) {
		if err := tpl.Execute(w, service); err != nil {
			web.RespondWithHtml(w, 500, err.Error())
			return
		}
	}
}

func (service *WebService) LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Infof("method=%s url=%s remoteAddr=%s referer=%s\n", req.Method, req.URL.String(), req.RemoteAddr, req.Referer())
		next.ServeHTTP(w, req)
	})
}

// staticFilesMiddleware returns the appropriate middleware depending on what
// mode we're running in.
func (service *WebService) staticFilesAssetProvider() (assetProvider web.AssetProvider) {
	/*switch config.Mode {
	case "prod":
	*/
	assetProvider = public.Asset
	/*

		case "dev": // Serve static files from disk.
			// Calculate the base public path only once to avoid unnecessary redundant computation.
	*/
	/*
		pieces := strings.Split(os.Args[0], "/")
		publicPath := strings.Join(append(pieces[0:len(pieces)-1], "public"), "/")
		log.Warnf("Dev disk-based static file asset provider activated, publicPath=%s", publicPath)
		assetProvider = web.DiskBasedAssetProvider(publicPath)
	*/
	/*
		default:
			panic(fmt.Errorf("unrecognized config.Mode: %v", config.Mode))
		}
		return
	*/
	return
}

/*func (service *WebService) txt(w http.ResponseWriter, req *http.Request) {
	url := hitch.Params(req).ByName("url")
	fmt.Println(url)

	if len(url) == 1 {
		web.RespondWithHtml(w, 200, `<html><head><title>TXT-Web</title></head><body>Welcome to TXT-Web!</body></html>`)
		return
	}

	if len(url) > 0 {
		url = url[1:]
	}
	url = normalizeURL(url)

	var (
		ch = asyncTXT(url)
		r  result
	)

	select {
	case r = <-ch:
	case <-time.After(AsyncRequestHandlerTimeout):
		r = result{"", fmt.Errorf("timed out after %v", AsyncRequestHandlerTimeout)}
	}

	if r.err != nil {
		web.RespondWithText(w, http.StatusInternalServerError, r.err.Error())
		return
	}
	web.RespondWithText(w, http.StatusOK, r.content)
}

type result struct {
	content string
	err     error
}

func asyncTXT(url string) chan result {
	ch := make(chan result)
	go func() {
		r := gorequest.New()
		resp, data, errs := r.Get(url).End()
		if err := errorlib.Merge(errs); err != nil {
			ch <- result{"", err}
			return
		}
		if resp.StatusCode/100 != 2 {
			err := fmt.Errorf("expected status code 2xx but actual=%v", resp.StatusCode)
			ch <- result{"", err}
			return
		}
		txt, err := html2text.FromString(data, html2text.Options{PrettyTables: true})
		if err != nil {
			ch <- result{"", err}
			return
		}
		ch <- result{txt, nil}
		return
	}()
	return ch
}

func normalizeURL(url string) string {
	if strings.HasPrefix(url, "//") {
		url = "http:" + url
	}
	if !strings.HasPrefix(strings.ToLower(url), "http://") && !strings.HasPrefix(strings.ToLower(url), "https://") {
		url = "http://" + url
	}
	return url
}*/
