package web

// go : generate bash -c "( test ${SKIP_JS}x = 1x || ( echo 'Compiling frontend assets' && cd .. && ( test ! -f package.json || ( npm install && ( command -v gulp 1>/dev/null 2>&1 && gulp build || node_modules/.bin/gulp build ) ) ) ) ) && echo 'Generating public package from go-bindata Asset: ../public/*' && mkdir -p ../public && cd ../public && go-bindata -pkg='public' -o public.go -ignore=public\\.go `find . -type d` && gofmt -w public.go || (echo 'oops.. there was a problem, you may need to install go-bindata or fix frontend stuff, see README.md' && exit 1)"

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"sync"

	"gigawatt.io/errorlib"
	"gigawatt.io/web"
	"gigawatt.io/web/route"
	"github.com/Masterminds/sprig"
	"github.com/hkwi/h2c"
	"github.com/nbio/hitch"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"jaytaylor.com/andromeda/crawler"
	"jaytaylor.com/andromeda/db"
	"jaytaylor.com/andromeda/domain"
	"jaytaylor.com/andromeda/web/public"
)

// TODO: Maybe rename WebService to Service, or relocate this pkg?

//var AsyncRequestHandlerTimeout time.Duration = 5 * time.Second

type Config struct {
	Addr    string
	DevMode bool
	Master  *crawler.Master
}

type WebService struct {
	listener net.Listener
	handler  http.Handler
	server   *http.Server
	mu       sync.Mutex

	Config *Config
	DB     db.Client
	hub    *Hub
}

func New(db db.Client, cfg *Config) *WebService {
	service := &WebService{
		Config: cfg,
		DB:     db,
		hub:    newHub(),
	}
	service.handler = service.activateRoutes().Handler()
	return service
}

func (service *WebService) Start() error {
	log.Info("WebService starting..")

	service.mu.Lock()
	defer service.mu.Unlock()

	if service.Config.Master == nil {
		return fmt.Errorf("Missing Master!!!!")
	}

	var err error

	if service.listener, err = net.Listen("tcp", service.Config.Addr); err != nil {
		return err
	}

	grpcServer := grpc.NewServer(grpc.MaxSendMsgSize(crawler.MaxMsgSize), grpc.MaxRecvMsgSize(crawler.MaxMsgSize))
	domain.RegisterRemoteCrawlerServiceServer(grpcServer, service.Config.Master)

	m := cmux.New(service.listener)
	grpcListener := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpListener := m.Match(cmux.HTTP2(), cmux.HTTP1Fast())

	service.server = &http.Server{
		Handler: &h2c.Server{
			Handler: service.handler,
		},
		// ReadTimeout:    ReadTimeout,
		// WriteTimeout:   WriteTimeout,
		// MaxHeaderBytes: ws.Options.MaxHeaderBytes,
		// TLSConfig:      ws.Options.TLSConfig,
		// ErrorLog:       ws.Options.ErrorLog,
	}

	g := errgroup.Group{}
	g.Go(func() error { return grpcServer.Serve(grpcListener) })
	g.Go(func() error { return service.server.Serve(httpListener) })
	g.Go(func() error { return m.Serve() })

	go func() {
		log.Infof("run server result: %s", g.Wait())
	}()

	// Attach and wire in crawler master event subscriber / broadcaster.
	evCh := make(chan string, 100)
	go func() {
		go service.hub.run()
		for {
			event, ok := <-evCh
			if !ok {
				return
			}
			service.hub.broadcast <- []byte(event)
		}
	}()
	service.Config.Master.Subscribe(evCh)

	log.WithField("addr", service.listener.Addr()).Info("WebService started")
	return nil
}

func (service *WebService) Stop() error {
	log.Info("WebService stopping..")

	service.mu.Lock()
	defer service.mu.Unlock()
	if service.listener == nil {
		return errorlib.NotRunningError
	}
	service.server.Close()
	log.Info("WebService stopped")
	return nil
}

func (service *WebService) Addr() net.Addr {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.listener != nil {
		return service.listener.Addr()
	}
	return nil
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
				// {"get", "/", service.tplAsset("index.tpl")},
				{"get", "/*package", service.pkg},
			},
		},
	}
	h := route.Activate(routes)
	return h
}

func (service *WebService) LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.WithField("method", req.Method).WithField("url", req.URL.String()).WithField("remote-addr", req.RemoteAddr).WithField("referer", req.Referer()).Info("http handler invoked")
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

func (service *WebService) tplAsset(asset string) func(w http.ResponseWriter, req *http.Request) {
	// TODO: consider moving this outside of closure?  any benefit?
	//log.Infof("NAMES: %s", public.AssetNames())
	content, err := service.staticFilesAssetProvider()(asset)
	if err != nil {
		panic(fmt.Errorf("problem with asset %q: %v", asset, err))
	}
	tpl := template.Must(template.New(asset).Funcs(sprig.FuncMap()).Parse(string(content)))

	return func(w http.ResponseWriter, req *http.Request) {
		if err := tpl.Execute(w, service); err != nil {
			web.RespondWithHtml(w, 500, err.Error())
			return
		}
	}
}

func (service *WebService) pkg(w http.ResponseWriter, req *http.Request) {
	pkgPath := strings.Trim(hitch.Params(req).ByName("package"), "/")
	if len(pkgPath) == 0 {
		service.tplAsset("index.tpl")(w, req)
		return
	}

	if pkgPath == "ws" {
		log.Debug("Passing off to serveWs handler")
		serveWs(service.hub, w, req)
		return
	}

	log.WithField("pkg-path", pkgPath).Info("GET /:package invoked")

	// Handle API request.
	// TODO: This is not good structure, clean it up with a proper mux.
	if strings.HasPrefix(pkgPath, "v1/") {
		pkgPath = strings.SplitN(pkgPath, "v1/", 2)[1]
		pkg, err := service.DB.Package(pkgPath)
		if err != nil {
			if err == db.ErrKeyNotFound {
				web.RespondWithHtml(w, 404, err.Error())
			} else {
				web.RespondWithHtml(w, 500, err.Error())
			}
			return
		}
		web.RespondWithJson(w, 200, pkg)
		return
	}

	const asset = "package.tpl"
	content, err := service.staticFilesAssetProvider()(asset)
	if err != nil {
		panic(fmt.Errorf("problem with asset %q: %v", asset, err))
	}
	tpl := template.Must(template.New(asset).Funcs(sprig.FuncMap()).Parse(string(content)))

	pkg, err := service.DB.Package(pkgPath)
	if err != nil {
		if err == db.ErrKeyNotFound {
			service.pkgFallback(w, req, pkgPath)
			return
		}
		if err == db.ErrKeyNotFound {
			web.RespondWithHtml(w, 404, err.Error())
		} else {
			web.RespondWithHtml(w, 500, err.Error())
		}
		return
	}

	if pkg.Path == pkgPath {
		if err := tpl.Execute(w, pkg); err != nil {
			web.RespondWithHtml(w, 500, err.Error())
			return
		}
	} else {
		service.subPkgFallback(w, req, pkgPath, pkg)
	}
}

func (service *WebService) pkgFallback(w http.ResponseWriter, req *http.Request, path string) {
	const asset = "list.tpl"
	content, err := service.staticFilesAssetProvider()(asset)
	if err != nil {
		panic(fmt.Errorf("problem with asset %q: %v", asset, err))
	}
	tpl := template.Must(template.New(asset).Funcs(sprig.FuncMap()).Parse(string(content)))

	// Try a prefix search.
	var pkgs map[string]*domain.Package
	if pkgs, err = service.DB.PathPrefixSearch(path); err == nil {
		if err := tpl.Execute(w, pkgs); err != nil {
			web.RespondWithHtml(w, 500, err.Error())
		}
		return
	}

	if err == db.ErrKeyNotFound {
		web.RespondWithHtml(w, 404, err.Error())
	} else {
		web.RespondWithHtml(w, 500, err.Error())
	}
}

type SubPackageContext struct {
	Path string
	Pkg  *domain.Package
	Sub  *domain.SubPackage
}

func (service *WebService) subPkgFallback(w http.ResponseWriter, req *http.Request, pkgPath string, pkg *domain.Package) {
	const asset = "subpackage.tpl"
	content, err := service.staticFilesAssetProvider()(asset)
	if err != nil {
		panic(fmt.Errorf("problem with asset %q: %v", asset, err))
	}
	tpl := template.Must(template.New(asset).Funcs(sprig.FuncMap()).Parse(string(content)))

	nPkgPath := domain.SubPackagePathNormalize(pkg.Path, pkgPath)
	if sub, ok := pkg.Data.SubPackages[nPkgPath]; ok {
		tplCtx := &SubPackageContext{
			Path: pkgPath,
			Pkg:  pkg,
			Sub:  sub,
		}
		if err := tpl.Execute(w, tplCtx); err != nil {
			web.RespondWithHtml(w, 500, err.Error())
		}
	} else {
		web.RespondWithHtml(w, 404, fmt.Sprintf(`%[1]q not found within <a href="/%[2]v">%[2]v</a>`, pkgPath, pkg.Path))
	}
}
