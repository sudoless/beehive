package beehive_cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
)

//gocyclo:ignore
func TestCORS(t *testing.T) {
	t.Parallel()

	config := &Config{
		AllowHosts:       []string{"example.com", "dashboard.example.com", "api.example.net"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           time.Second * 3600,
	}

	ok := &beehive.DefaultResponder{
		Message: []byte("ok"),
		Status:  http.StatusOK,
	}

	router := beehive.NewRouter()
	corsGroup := config.Apply(router)
	corsGroup.Handle("GET", "/foo/bar", func(_ *beehive.Context) beehive.Responder {
		return ok
	})

	t.Run("pass", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}

		h := w.Header()
		if h.Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("expected Access-Control-Allow-Origin %s, got %s", "https://example.com", h.Get("Access-Control-Allow-Origin"))
		}
		if h.Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE" {
			t.Errorf("expected Access-Control-Allow-Methods %s, got %s", "GET, POST, PUT, DELETE", h.Get("Access-Control-Allow-Methods"))
		}
		if h.Get("Access-Control-Allow-Headers") != "*" {
			t.Errorf("expected Access-Control-Allow-Headers %s, got %s", "*", h.Get("Access-Control-Allow-Headers"))
		}
		if h.Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials %s, got %s", "true", h.Get("Access-Control-Allow-Credentials"))
		}
		if h.Get("Access-Control-Max-Age") != "3600" {
			t.Errorf("expected Access-Control-Max-Age %s, got %s", "3600", h.Get("Access-Control-Max-Age"))
		}
	})
	t.Run("deny", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("Origin", "https://api.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, w.Code)
		}
	})
	t.Run("bad origin url", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("Origin", "://://://")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, w.Code)
		}
	})
	t.Run("no origin, pass", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("Origin", "")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}
	})
	t.Run("pass OPTIONS", func(t *testing.T) {
		r := httptest.NewRequest("OPTIONS", "/foo/bar", nil)
		r.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status code %d, got %d", http.StatusNoContent, w.Code)
		}

		h := w.Header()
		if h.Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("expected Access-Control-Allow-Origin %s, got %s", "https://example.com", h.Get("Access-Control-Allow-Origin"))
		}
		if h.Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE" {
			t.Errorf("expected Access-Control-Allow-Methods %s, got %s", "GET, POST, PUT, DELETE", h.Get("Access-Control-Allow-Methods"))
		}
		if h.Get("Access-Control-Allow-Headers") != "*" {
			t.Errorf("expected Access-Control-Allow-Headers %s, got %s", "*", h.Get("Access-Control-Allow-Headers"))
		}
		if h.Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials %s, got %s", "true", h.Get("Access-Control-Allow-Credentials"))
		}
		if h.Get("Access-Control-Max-Age") != "3600" {
			t.Errorf("expected Access-Control-Max-Age %s, got %s", "3600", h.Get("Access-Control-Max-Age"))
		}
	})
	t.Run("deny OPTIONS", func(t *testing.T) {
		r := httptest.NewRequest("OPTIONS", "/foo/bar", nil)
		r.Header.Set("Origin", "https://api.example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, w.Code)
		}
	})
}
