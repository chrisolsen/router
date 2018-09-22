package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_SetHeader(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mw := SetHeader("foo", "bar")
	mw(w, r)

	if w.Header().Get("foo") != "bar" {
		t.Error("Header not set")
		return
	}
}

func Test_SetHeaders(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mw := SetHeaders(map[string]string{
		"foo":  "bar",
		"boom": "blam",
	})

	mw(w, r)

	if w.Header().Get("foo") != "bar" {
		t.Error("Foo header not set")
		return
	}

	if w.Header().Get("boom") != "blam" {
		t.Error("Boom header not set")
		return
	}
}
