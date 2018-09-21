# Router [![Build Status](https://travis-ci.org/chrisolsen/router.svg?branch=master)](https://travis-ci.org/chrisolsen/router) [![Coverage Status](https://coveralls.io/repos/github/chrisolsen/router/badge.svg?branch=master)](https://coveralls.io/github/chrisolsen/router?branch=master)

Provides simple routing capabilities. 

```Go
func main() {
    rr := router.New("/")

    rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello world")
    })

    if err := http.ListenAndServe(":80", rr); err != nil {
        fmt.Printf("failed to start server: %v", err.Error())
    }
}
```

## Subroute
```Go
func main() {
    rr := router.New("/")
    subrr := rr.SubRouter("/admin")

    rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello world")
    })

    subrr.Get("/payroll", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "in /admin/payroll")
    })

    http.ListenAndServe(":80", rr)
}
```

## Middleware
```Go

type tokenMiddleware struct { }

func (t tokenMiddleware) SetToken(r *http.Request) {
    token := generateToken()
    c2 := context.WithValue(r.Context(), "key", token)
    router.BindContext(c2, r)
}

func (t tokenMiddleware) Token(c context.Context) string {
    return c.Value("key").(string)  // might want to be more cautious than this
}

func main() {
    rr := router.New("/")

    rr.Before(func (w http.ResponseWriter, r *http.Request) {
        tokenMiddleware.SetToken(r)
    })

    rr.Get("/", func(w http.ResponseWriter, r *http.Request) {
        token := tokenMiddleware.Token(r.Context())
        fmt.Fprintf(w, "TOKEN: %s", token)
    })

    http.ListenAndServe(":80", rr)
}
```



## Extract URL params

```Go
rr.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
    params := router.Params(r.Context())
    fmt.Fprintf(w, "Hey %s", params["name"])
})
```

## Use handlers
```Go
type usersHandler struct {
    svc aService
}
func (h usersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.svc.doSomething()
}

rr.Handle("GET", "/users", usersHandler{svc: Service.New()})
```

## 404 handling
```Go
rr.NotFound(func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Not found")
})
```

## Wildcard params
```Go
// GET: /hello/go/programmer
rr.Get("/hello/*", func(w http.ResponseWriter, r *http.Request) {
    wildcard := router.Param(r.Context(), "*") // => "go/programmer"
    vals = strings.Split(wildcard, "/")
    fmt.Fprintf(w, "Hello %s %s", vals[0], vals[1])
})
```