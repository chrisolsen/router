package router

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

var paramsCtxKey = ctxKey("params")

type props struct {
	fn      http.HandlerFunc
	handler http.Handler
}

// Route is a route
type Route struct {
	path   string
	method string
}

// New creates a new router, allowing for the setup of route handling
func New(path string) Router {
	if len(path) == 0 {
		path = "/"
	}
	return Router{
		basePath: path,
		routes:   make(map[Route]props),
	}
}

// BindContext links the new context with the request to allow for any context values
// to be available later in the chain
func BindContext(c context.Context, r *http.Request) {
	*r = *r.WithContext(c)
}

// HaltRequest is most commonly called with the middleware to stop the middleware chain
// from continuing as well as prevent the final handler from being run
func HaltRequest(r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	BindContext(ctx, r)
	cancel()
}

// Params retrieves the url parameters matched
func Params(c context.Context) map[string]string {
	switch c.Value(paramsCtxKey).(type) {
	case map[string]string:
		return c.Value(paramsCtxKey).(map[string]string)
	default:
		return map[string]string{}
	}
}

// Param gets the names url param
func Param(c context.Context, key string) string {
	return Params(c)[key]
}

// Router is a custom mux that allows for url parameter to be extracted from the path
type Router struct {
	basePath        string
	routes          map[Route]props
	subRouters      []*Router
	notFoundHandler http.HandlerFunc

	mw []http.HandlerFunc
}

// Before injects the passed in handler functions into the handler chain
func (r *Router) Before(fns ...http.HandlerFunc) {
	r.mw = append(r.mw, fns...)
}

// Run executes the handler chain, followed by the final http handler passed in
func (r Router) run(last http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		for _, fn := range r.mw {
			fn(w, req)
			if req.Context().Err() != nil {
				return
			}
		}
		last(w, req)
	}
}

func (r Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rr := r.findMatchingRouter(req.URL.Path)
	for rparams, props := range rr.routes {
		// handler accepts all method types
		if rparams.method != req.Method && props.handler == nil {
			continue
		}
		fullPattern := strings.Join([]string{r.basePath, rparams.path}, "")
		if ok, params := matches(fullPattern, req.URL.Path); ok {
			var handler http.HandlerFunc
			if props.fn != nil {
				handler = props.fn
			} else if props.handler != nil {
				handler = props.handler.ServeHTTP
			}

			rr.Before(setURLParams(req, params))
			rr.run(handler)(w, req)

			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	if r.notFoundHandler != nil {
		r.notFoundHandler(w, req)
	}
}

// HandleFunc allows the handler to be called when the path matches the request's url path
func (r Router) HandleFunc(method, path string, fn http.HandlerFunc) {
	r.bindRoute(method, path, props{fn: fn})
}

// SubRouter creates a child router with a custom base path
func (r *Router) SubRouter(path string) *Router {
	var basePath string
	if r.basePath != "/" {
		basePath = r.basePath
	}
	sub := Router{
		basePath: basePath + path,
		routes:   make(map[Route]props),
	}
	r.subRouters = append(r.subRouters, &sub)
	return &sub
}

func (r Router) bindRoute(method, path string, p props) {
	r.routes[Route{method: method, path: path}] = p
}

// Get handles GET requests
func (r Router) Get(path string, fn http.HandlerFunc) {
	r.bindRoute(http.MethodGet, path, props{fn: fn})
}

// Post handles POST requests
func (r Router) Post(path string, fn http.HandlerFunc) {
	r.bindRoute(http.MethodPost, path, props{fn: fn})
}

// Put handles PUT requests
func (r Router) Put(path string, fn http.HandlerFunc) {
	r.bindRoute(http.MethodPut, path, props{fn: fn})
}

// Delete handles DELETE requests
func (r Router) Delete(path string, fn http.HandlerFunc) {
	r.bindRoute(http.MethodDelete, path, props{fn: fn})
}

// Patch handles PATCH requests
func (r Router) Patch(path string, fn http.HandlerFunc) {
	r.bindRoute(http.MethodPatch, path, props{fn: fn})
}

// Handle foo
func (r Router) Handle(path string, h http.Handler) {
	r.routes[Route{path: path}] = props{handler: h}
}

// NotFound allows for a custom 404 handler to be set
func (r *Router) NotFound(h http.HandlerFunc) {
	r.notFoundHandler = h
}

// Finds the matching router
func (r Router) findMatchingRouter(urlPath string) *Router {
	for _, child := range r.subRouters {
		if r := child.findMatchingRouter(urlPath); r != nil {
			return r
		}
	}
	if strings.Index(urlPath, r.basePath) == 0 {
		return &r
	}
	return nil
}

func matches(pattern, path string) (bool, map[string]string) {
	wildcard := strings.Contains(pattern, "*")
	if strings.Index(pattern, ":") == -1 && !wildcard {
		return strings.Trim(path, "/") == strings.Trim(pattern, "/"), nil
	}

	pathParts, patternParts := slicePath(path), slicePath(pattern)

	if wildcard {
		if len(pathParts) < len(patternParts) {
			return false, nil
		}
		return true, map[string]string{
			"*": strings.Join(pathParts[len(patternParts)-1:], "/"),
		}
	}

	patternPartCount, pathPartCount := len(patternParts), len(pathParts)
	if pathPartCount != patternPartCount {
		return false, nil
	}

	// check parts
	for i := 0; i < patternPartCount; i++ {
		pathPart, patternPart := pathParts[i], patternParts[i]
		if patternPart[0] == ':' {
			continue
		}
		if pathPart != patternPart {
			return false, nil
		}
	}

	// extract pattern params
	params := make(map[string]string)
	for i, part := range patternParts {
		if part[0] == ':' {
			params[part[1:]] = pathParts[i]
		}
	}

	return true, params
}

func slicePath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func setURLParams(r *http.Request, params map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := context.WithValue(r.Context(), paramsCtxKey, params)
		*r = *r.WithContext(c)
	}
}
