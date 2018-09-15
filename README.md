# Router

Provides simple routing capabilities. 

```Go
func main() {
    rr := router.New("/")

    rr.HandleFunc("GET", "/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello world")
    })

    if err := http.ListenAndServe(":80", rr); err != nil {
        fmt.Printf("failed to start server: %v", err.Error())
    }
}
```

## Extract URL params

```Go
rr.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
    params := router.Params(r.Context())
    fmt.Fprintf(w, "Hey %s", params["name"])
})
```

## Use http.Handler
```Go
rr.Handle("GET", "/users", usersHandler{})
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
rr.HandleFunc("GET", "/hello/*", func(w http.ResponseWriter, r *http.Request) {
    wildcard := router.Param(r.Context(), "*") // => "go/programmer"
    vals = strings.Split(wildcard, "/")
    fmt.Fprintf(w, "Hello %s %s", vals[0], vals[1])
})
```