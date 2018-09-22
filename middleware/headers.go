package middleware

import (
	"net/http"
)

// SetHeader sets the response header to the key and value provided
func SetHeader(key, value string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(key, value)
	}
}

// SetHeaders sets the response header to the keys/values provided
func SetHeaders(vals map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for key, val := range vals {
			w.Header().Set(key, val)
		}
	}
}
