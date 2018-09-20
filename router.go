package router

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

var paramsCtxKey = ctxKey("params")

type props struct {
	method  string
	fn      http.HandlerFunc
	handler http.Handler
}

// Router is a custom mux that allows for url parameter to be extracted from the path
type Router struct {
	basePath        string
	routes          map[string]props
	subRouters      []*Router
	notFoundHandler http.HandlerFunc

	mw []http.HandlerFunc
}

// Before injects the passed in handler functions into the handler chain
func (r *Router) Before(fns ...http.HandlerFunc) {
	r.mw = append(r.mw, fns...)
}

// Run executes the handler chain, followed by the final http handler passed in
func (r Router) Run(last http.HandlerFunc) http.HandlerFunc {
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

// New creates a new router, allowing for the setup of route handling
func New(path string) Router {
	if len(path) == 0 {
		path = "/"
	}
	return Router{
		basePath: path,
		routes:   make(map[string]props),
	}
}

func (r Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rr := r.findMatchingRouter(req.URL.Path)
	for path, props := range rr.routes {
		// handler accepts all method types
		if props.method != req.Method && props.handler == nil {
			continue
		}
		fullPattern := strings.Join([]string{r.basePath, path}, "")
		if ok, params := matches(fullPattern, req.URL.Path); ok {
			c := context.WithValue(req.Context(), paramsCtxKey, params)
			req = req.WithContext(c)
			if props.fn != nil {
				props.fn(w, req)
			} else if props.handler != nil {
				props.handler.ServeHTTP(w, req)
			}
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
	r.routes[path] = props{method: method, fn: fn}
}

// SubRouter creates a child router with a custom base path
func (r *Router) SubRouter(path string) *Router {
	var basePath string
	if r.basePath != "/" {
		basePath = r.basePath
	}
	sub := Router{
		basePath: basePath + path,
		routes:   make(map[string]props),
	}
	r.subRouters = append(r.subRouters, &sub)
	return &sub
}

// Get handles GET requests
func (r Router) Get(path string, fn http.HandlerFunc) {
	r.routes[path] = props{method: http.MethodGet, fn: fn}
}

// Post handles POST requests
func (r Router) Post(path string, fn http.HandlerFunc) {
	r.routes[path] = props{method: http.MethodPost, fn: fn}
}

// Put handles PUT requests
func (r Router) Put(path string, fn http.HandlerFunc) {
	r.routes[path] = props{method: http.MethodPut, fn: fn}
}

// Delete handles DELETE requests
func (r Router) Delete(path string, fn http.HandlerFunc) {
	r.routes[path] = props{method: http.MethodDelete, fn: fn}
}

// Patch handles PATCH requests
func (r Router) Patch(path string, fn http.HandlerFunc) {
	r.routes[path] = props{method: http.MethodPatch, fn: fn}
}

// Handle foo
func (r Router) Handle(path string, h http.Handler) {
	r.routes[path] = props{handler: h}
}

// NotFound allows for a custom 404 handler to be set
func (r *Router) NotFound(h http.HandlerFunc) {
	r.notFoundHandler = h
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
