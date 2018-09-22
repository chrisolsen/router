package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInitialRequestWithMissingHeaders(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	mw := BasicAuth(nil)
	mw(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Error("Invalid response status: ", w.Code)
		return
	}

	header := w.Header().Get("www-authenticate")
	if len(header) == 0 {
		t.Error("missing www-authenticate header")
		return
	}

	if strings.Index(header, "Basic") != 0 {
		t.Error("Missing `Basic` in header")
		return
	}
}

func TestInvalidCredentials(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	r.Header.Add("Authorization", `Basic Zm9vOmJhcg==`)

	mw := BasicAuth(func(c context.Context, user, password string) bool {
		return false
	})
	mw(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Error("Invalid response status: ", w.Code)
		return
	}
}

func TestBadEncoding(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	r.Header.Add("Authorization", `Basic Zm9vOmJhcgZZZ`)

	mw := BasicAuth(func(c context.Context, user, password string) bool {
		return true
	})
	mw(w, r)

	if w.Code != http.StatusBadRequest {
		t.Error("Invalid response status: ", w.Code)
		return
	}
}

func TestBadCredentials(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	val := base64.RawStdEncoding.EncodeToString([]byte("invalid_creds"))
	r.Header.Add("Authorization", `Basic `+val)

	mw := BasicAuth(func(c context.Context, user, password string) bool {
		return true
	})
	mw(w, r)

	if w.Code != http.StatusBadRequest {
		t.Error("Invalid response status: ", w.Code)
		return
	}
}

func TestValidCredentials(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.Header.Add("Authorization", `Basic Zm9vOmJhcg==`)

	mw := BasicAuth(func(c context.Context, user, password string) bool {
		return true
	})
	mw(w, r)

	if w.Code != http.StatusOK {
		t.Error("Invalid response status: ", w.Code)
		return
	}
}
