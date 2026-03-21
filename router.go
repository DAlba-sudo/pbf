package pbf

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
)

var (
	logger = slog.New(slog.NewJSONHandler(log.Writer(), nil))
)

type Handler func(http.ResponseWriter, *http.Request) error

type Router struct {
	Address string
	Port    int
	mux     *http.ServeMux

	middleware []Handler
	routes     map[string](map[string]Handler)
}

type RouteOptions struct {
	Endpoint string
	Method   string

	Handler Handler
}

func (r *Router) Add(opt RouteOptions) {
	_, ok := r.routes[opt.Endpoint]
	if !ok {
		r.routes[opt.Endpoint] = make(map[string]Handler)
	}

	r.routes[opt.Endpoint][opt.Method] = opt.Handler
}

func (r *Router) SetMiddleware(h Handler) {
	r.middleware = append(r.middleware, h)
}

func CreateRouter() *Router {
	r := &Router{
		mux:        http.NewServeMux(),
		middleware: []Handler{},
		routes:     make(map[string](map[string]Handler)),
	}

	r.mux.HandleFunc("/", r.Runner)

	return r
}

func cleanPath(path string) string {
	p := strings.TrimRight(path, "/")
	return p
}

func (r Router) Runner(w http.ResponseWriter, req *http.Request) {
	path := cleanPath(req.URL.Path)

	for _, mw := range r.middleware {
		err := mw(w, req)
		if err != nil {
			logger.Error("middleware error", "path", path, "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	methods, ok := r.routes[path]
	if !ok {
		logger.Warn("route not found", "path", path)
		http.NotFound(w, req)
		return
	}

	handler, ok := methods[req.Method]
	if !ok {
		logger.Warn("method not allowed", "path", path, "method", req.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

	err := handler(w, req)
	if err != nil {
		logger.Error("handler error", "path", path, "method", req.Method, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Router) Start() error {
	logger.Info("starting router", "address", r.Address, "port", r.Port)

	return http.ListenAndServe(fmt.Sprintf("%s:%d", r.Address, r.Port), r.mux)
}
