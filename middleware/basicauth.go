package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/chrisolsen/router"
)

// BasicAuth performs the authentication using the passed in auth function
func BasicAuth(auth func(c context.Context, name, password string) bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) == 0 {
			w.Header().Set("WWW-Authenticate", `Basic realm=""`)
			w.WriteHeader(http.StatusUnauthorized)
			router.HaltRequest(r)
			return
		}

		// `=` represents a space and therefore needs to be trimmed
		raw := strings.TrimRight(authHeader[len("Basic "):], "=")
		input, err := base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			router.HaltRequest(r)
			return
		}

		parts := strings.Split(string(input), ":")
		if len(parts) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			router.HaltRequest(r)
			return
		}

		if !auth(r.Context(), parts[0], parts[1]) {
			w.WriteHeader(http.StatusUnauthorized)
			router.HaltRequest(r)
			return
		}
	}
}
