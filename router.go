package router

import (
	"context"
	"net/http"
	"strings"
)

var paramsCtxKey = ctxKey{"params"}

type handlerFunc = func(http.ResponseWriter, *http.Request)

type ctxKey struct {
	name string
}

type props struct {
	method  string
	fn      handlerFunc
	handler http.Handler
}

// Router is a custom mux that allows for url parameter to be extracted from the path
type Router struct {
	basePath        string
	routes          map[string]props
	notFoundHandler handlerFunc
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
	for path, props := range r.routes {
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
func (r Router) HandleFunc(method, path string, fn handlerFunc) {
	r.routes[path] = props{method: method, fn: fn}
}

// Get handles GET requests
func (r Router) Get(path string, fn handlerFunc) {
	r.routes[path] = props{method: http.MethodGet, fn: fn}
}

// Post handles POST requests
func (r Router) Post(path string, fn handlerFunc) {
	r.routes[path] = props{method: http.MethodPost, fn: fn}
}

// Put handles PUT requests
func (r Router) Put(path string, fn handlerFunc) {
	r.routes[path] = props{method: http.MethodPut, fn: fn}
}

// Delete handles DELETE requests
func (r Router) Delete(path string, fn handlerFunc) {
	r.routes[path] = props{method: http.MethodDelete, fn: fn}
}

// Patch handles PATCH requests
func (r Router) Patch(path string, fn handlerFunc) {
	r.routes[path] = props{method: http.MethodPatch, fn: fn}
}

// Handle foo
func (r Router) Handle(path string, h http.Handler) {
	r.routes[path] = props{handler: h}
}

// NotFound allows for a custom 404 handler to be set
func (r *Router) NotFound(h handlerFunc) {
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
