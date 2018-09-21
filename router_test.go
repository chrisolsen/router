package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type testHandler struct {
	status int
	body   string
}

func (th testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(th.status)
	w.Write([]byte(th.body))
}

func TestRouteMatching(t *testing.T) {

	tests := map[string]map[string]bool{
		"/users/:name": map[string]bool{
			"/":            false,
			"/users":       false,
			"/users/":      false,
			"/users/123":   true,
			"/users/john":  true,
			"/users/john/": true,
		},
		"/projects/:id/approve": map[string]bool{
			"/":                      false,
			"/projects":              false,
			"/projects/":             false,
			"/projects/123":          false,
			"/projects/123/":         false,
			"/projects/123/approve":  true,
			"/projects/123/approve/": true,
			"/projects/123/deny":     false,
		},
		"/users/*": map[string]bool{
			"/users":       false,
			"/users/a":     true,
			"/users/a/b":   true,
			"/users/a/b/c": true,
		},
	}

	for pattern, urls := range tests {
		for url, shouldMatch := range urls {
			matched, _ := matches(pattern, url)
			if shouldMatch && !matched {
				t.Errorf("%s should match %s", pattern, url)
			}
			if !shouldMatch && matched {
				t.Errorf("%s should not match %s", pattern, url)
			}
		}
	}
}

func TestNewRouter(t *testing.T) {
	tests := []struct {
		expectedPath string
		path         string
	}{
		{expectedPath: "/someUrl", path: "/someUrl"},
		{expectedPath: "/", path: "/"},
		{expectedPath: "/", path: ""},
	}

	for _, test := range tests {
		r := New(test.path)
		if r.basePath != test.expectedPath {
			t.Errorf("basePath set incorrectly '%s' != '%s'", r.basePath, test.expectedPath)
		}
		if r.routes == nil {
			t.Error("routes not initialized")
		}
	}

}

func TestServeHTTPHandlers(t *testing.T) {

	tests := []struct {
		handlerMethod string
		handlerPath   string
		handlerFunc   http.HandlerFunc
		handler       http.Handler

		calledMethod     string
		calledPath       string
		expectedStatus   int
		expectedResponse string
	}{
		{
			handlerPath:   "/",
			handlerMethod: "GET",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte("test 1"))
			},
			calledMethod:     "GET",
			calledPath:       "/",
			expectedStatus:   200,
			expectedResponse: "test 1",
		},
		{
			handlerPath:   "/",
			handlerMethod: "POST",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(201)
				w.Write([]byte("test 2"))
			},
			calledMethod:     "POST",
			calledPath:       "/",
			expectedStatus:   201,
			expectedResponse: "test 2",
		},
		{
			handlerPath:   "/",
			handlerMethod: "PUT",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(204)
				w.Write([]byte("test 3"))
			},
			calledMethod:     "PUT",
			calledPath:       "/",
			expectedStatus:   204,
			expectedResponse: "test 3",
		},
		{
			handlerPath:   "/",
			handlerMethod: "DELETE",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte("test 4"))
			},
			calledMethod:     "DELETE",
			calledPath:       "/",
			expectedStatus:   200,
			expectedResponse: "test 4",
		},
		{
			handlerPath:   "/",
			handlerMethod: "PATCH",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte("test 5"))
			},
			calledMethod:     "PATCH",
			calledPath:       "/",
			expectedStatus:   200,
			expectedResponse: "test 5",
		},
		{
			handlerPath: "/",
			handler: testHandler{
				status: 200,
				body:   "test 6",
			},
			calledMethod:     "PATCH",
			calledPath:       "/",
			expectedStatus:   200,
			expectedResponse: "test 6",
		},
	}

	for _, test := range tests {
		router := New("/")
		if test.handlerFunc != nil {
			router.HandleFunc(test.handlerMethod, test.handlerPath, test.handlerFunc)
		} else if test.handler != nil {
			router.Handle(test.handlerPath, test.handler)
		}

		req, err := http.NewRequest(test.calledMethod, test.calledPath, nil)
		if err != nil {
			t.Error("failed to create request")
			return
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != test.expectedStatus {
			t.Errorf("Invalid status code %d != %d", rec.Code, test.expectedStatus)
			return
		}
		if rec.Body.String() != test.expectedResponse {
			t.Errorf("Invalid body %s != %s", rec.Body.String(), test.expectedResponse)
			return
		}
	}
}

func TestMiddleware(t *testing.T) {
	testKey := ctxKey("test")
	ch := make(chan bool)

	tests := []struct {
		desc       string
		err        string
		middleware []http.HandlerFunc
		handler    http.HandlerFunc
	}{
		{
			desc: "handlerFunc is called",
			err:  "context data not set correctly",
			middleware: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					ctx := context.WithValue(r.Context(), testKey, "bar")
					BindContext(ctx, r)
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				data := r.Context().Value(testKey)
				switch data.(type) {
				case string:
					ch <- (data.(string) == "bar")
				default:
					ch <- false
				}
			},
		},
		{
			desc: "error exists in middleware, so later middleware should not be called",
			err:  "final method should not be called",
			middleware: []http.HandlerFunc{
				func(w http.ResponseWriter, r *http.Request) {
					HaltRequest(r)
				},
				func(w http.ResponseWriter, r *http.Request) {
					ch <- false // should not make it here
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				ch <- false // should not make it here
			},
		},
	}

	for _, test := range tests {
		rr := New("/")
		rr.Before(test.middleware...)
		rr.Get("/", test.handler)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go (func() {
			select {
			case val := <-ch:
				if !val {
					t.Error(test.err)
				}
				wg.Done()
			// wait for any channel data
			case <-time.Tick(100 * time.Millisecond):
				wg.Done()
			}
		})()

		r, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Error("failed to create request")
			return
		}
		w := httptest.NewRecorder()

		rr.run(test.handler)(w, r)
		wg.Wait()
	}
}

func TestNotFound(t *testing.T) {
	tests := []struct {
		handlerMethod string
		handlerPath   string
		handlerFunc   http.HandlerFunc

		calledMethod     string
		calledPath       string
		expectedStatus   int
		expectedResponse string
		notFoundHandler  http.HandlerFunc
	}{
		{
			// validate the method matches
			handlerPath:   "/",
			handlerMethod: "GET",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
			calledMethod:   "GET",
			calledPath:     "/invalid_path",
			expectedStatus: 404,
		},
		{
			// validate the method matches
			handlerPath:   "/",
			handlerMethod: "GET",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			},
			calledMethod:   "POST",
			calledPath:     "/",
			expectedStatus: 404,
		},
		{
			// validate the custom 404 handler is run
			handlerPath:      "/",
			handlerMethod:    "GET",
			calledMethod:     "POST",
			calledPath:       "/",
			expectedStatus:   404,
			expectedResponse: "not found yo",
			notFoundHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not found yo"))
			},
		},
	}

	for _, test := range tests {
		router := New("/")
		router.HandleFunc(test.handlerMethod, test.handlerPath, test.handlerFunc)
		router.NotFound(test.notFoundHandler)

		req, err := http.NewRequest(test.calledMethod, test.calledPath, nil)
		if err != nil {
			t.Error("failed to create request")
			return
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != test.expectedStatus {
			t.Errorf("Invalid status code %d != %d", rec.Code, test.expectedStatus)
			return
		}

		if rec.Body.String() != test.expectedResponse {
			t.Errorf("Invalid response %s != %s", rec.Body.String(), test.expectedResponse)
			return
		}
	}
}

func TestUrlParamsAreExtractedIntoContext(t *testing.T) {
	tests := []struct {
		isSubRoute  bool
		handlerFunc http.HandlerFunc
		method      string
		pathMatcher string
		path        string
	}{
		{
			method:      "GET",
			pathMatcher: "/users/:id",
			path:        "/users/123",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				id := Param(r.Context(), "id")
				if id != "123" {
					t.Errorf("failed to extract url param: %s expected, got %s", "123", id)
				}
			},
		},
		{
			method:      "GET",
			pathMatcher: "/users/:userid/tasks/:taskid",
			path:        "/users/123/tasks/456",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				userid := Param(r.Context(), "userid")
				if userid != "123" {
					t.Errorf("failed to extract url param: %s expected, got %s", "123", userid)
				}
				taskid := Param(r.Context(), "taskid")
				if taskid != "456" {
					t.Errorf("failed to extract url param: %s expected, got %s", "456", taskid)
				}
			},
		},
		{
			isSubRoute:  true,
			method:      "GET",
			pathMatcher: "/admin/tasks/:taskid",
			path:        "/admin/tasks/456",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				taskid := Param(r.Context(), "taskid")
				if taskid != "456" {
					t.Errorf("failed to extract url param: %s expected, got %s", "456", taskid)
				}
			},
		},
	}

	for _, test := range tests {
		router := New("/")
		subrouter := router.SubRouter("/admin")

		if test.isSubRoute {
			subrouter.HandleFunc(test.method, test.pathMatcher, test.handlerFunc)
		} else {
			router.HandleFunc(test.method, test.pathMatcher, test.handlerFunc)
		}

		req, err := http.NewRequest(test.method, test.path, nil)
		if err != nil {
			t.Error("failed to create request")
			return
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("200OK not received: Got %d", rec.Code)
			return
		}
	}
}

// validate method helper functions (Get, Post, etc)
func TestGetHelper(t *testing.T) {
	rr := New("/")
	rr.Get("/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodGet {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}

func TestPostHelper(t *testing.T) {
	rr := New("/")
	rr.Post("/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodPost {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}

func TestPuttHelper(t *testing.T) {
	rr := New("/")
	rr.Put("/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodPut {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}
func TestDeleteHelper(t *testing.T) {
	rr := New("/")
	rr.Delete("/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodDelete {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}

func TestPatchHelper(t *testing.T) {
	rr := New("/")
	rr.Patch("/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodPatch {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}

func TestHandlerFuncHelper(t *testing.T) {
	rr := New("/")
	rr.HandleFunc("GET", "/foo", func(w http.ResponseWriter, r *http.Request) {})

	route := rr.routes["/foo"]
	if route.method != http.MethodGet {
		t.Error("method not set")
		return
	}
	if route.fn == nil {
		t.Error("handler is nil")
		return
	}
}

func TestHandle(t *testing.T) {
	rr := New("/")
	rr.Handle("/foo", testHandler{})

	route := rr.routes["/foo"]
	if route.handler == nil {
		t.Error("handler is nil")
		return
	}
}

// validate Params
func TestParams(t *testing.T) {
	var paramsCtxKey = ctxKey("params")

	params := map[string]string{
		"foo": "bar",
	}
	p := context.Background()
	c := context.WithValue(p, paramsCtxKey, params)

	data := Params(c)

	if data["foo"] == "" {
		t.Error("param not being set")
		return
	}
	if data["foo"] != params["foo"] {
		t.Error("params don't match")
	}
}

// validate empty params
func TestEmptyParams(t *testing.T) {
	c := context.Background()

	if p := Param(c, "foo"); p != "" {
		t.Error("empty param should be returned")
		return
	}
}

// validate slicePath
func TestSlicePath(t *testing.T) {
	tests := []struct {
		given    string
		expected []string
	}{
		{given: "foo/bar", expected: []string{"foo", "bar"}},
		{given: "/foo/bar", expected: []string{"foo", "bar"}},
		{given: "/foo/bar/", expected: []string{"foo", "bar"}},
		{given: "/", expected: []string{""}},
	}

	for _, test := range tests {
		result := slicePath(test.given)
		if result == nil {
			t.Error("slice is nil")
			return
		}
		if len(result) != len(test.expected) {
			t.Error("length doesn't match")
			return
		}
		for i, item := range test.expected {
			if item != result[i] {
				t.Errorf("item doesn't match: %s != %s", item, result[i])
				return
			}
		}
	}
}

func TestSubRouterMatching(t *testing.T) {
	r := New("/")
	s := r.SubRouter("/admin")
	ss := s.SubRouter("/payroll")

	rtests := []struct {
		path    string
		matches bool
	}{
		{"/", true},
		{"/a", true},
		{"/a/b", true},
		{"/admin/a", false},
		{"/admin/payroll/a", false},
		{"/payroll", true},
	}
	for _, test := range rtests {
		matches := r.findMatchingRouter(test.path).basePath == r.basePath
		if test.matches != matches {
			t.Errorf("failed to match: %s => %s", test.path, r.basePath)
		}
	}

	stests := []struct {
		path    string
		matches bool
	}{
		{"/", false},
		{"/a", false},
		{"/a/b", false},
		{"/admin/a", true},
		{"/admin/payroll/a", false},
		{"/payroll", false},
	}
	for _, test := range stests {
		matches := r.findMatchingRouter(test.path).basePath == s.basePath
		if test.matches != matches {
			t.Errorf("failed to match: %s => %s", test.path, s.basePath)
		}
	}

	sstests := []struct {
		path    string
		matches bool
	}{
		{"/", false},
		{"/a", false},
		{"/a/b", false},
		{"/admin/a", false},
		{"/admin/payroll/a", true},
		{"/payroll", false},
	}
	for _, test := range sstests {
		matchingRouter := r.findMatchingRouter(test.path)
		matches := matchingRouter.basePath == ss.basePath
		if test.matches != matches {
			t.Errorf("failed to match: %s => %s", test.path, ss.basePath)
		}
	}
}
