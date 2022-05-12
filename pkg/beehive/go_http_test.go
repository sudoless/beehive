package beehive

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrapHttpHandlerFunc(t *testing.T) {
	t.Parallel()

	t.Run("simple handler", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("Hello, World!"))
		}

		router := NewRouter()
		router.Handle("GET", "/foo", WrapHttpHandlerFunc(h))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected status code %d, got %d", http.StatusTeapot, w.Code)
		}

		if w.Body.String() != "Hello, World!" {
			t.Errorf("expected body %q, got %q", "Hello, World!", w.Body.String())
		}
	})
	t.Run("no response written", func(t *testing.T) {
		h := func(w http.ResponseWriter, r *http.Request) {
			// noop
		}

		router := NewRouter()
		router.Handle("GET", "/foo", WrapHttpHandlerFunc(h))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "" {
			t.Errorf("expected body %q, got %q", "", w.Body.String())
		}
	})
}

type testHandler struct{}

func (t testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/simple":
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("Hello, World!"))
	case "/noop":
		// noop
	}
}

func TestWrapHttpHandler(t *testing.T) {
	t.Parallel()

	t.Run("simple handler", func(t *testing.T) {
		h := testHandler{}

		router := NewRouter()
		router.Handle("GET", "/simple", WrapHttpHandler(h))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/simple", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected status code %d, got %d", http.StatusTeapot, w.Code)
		}

		if w.Body.String() != "Hello, World!" {
			t.Errorf("expected body %q, got %q", "Hello, World!", w.Body.String())
		}
	})
	t.Run("no response written", func(t *testing.T) {
		h := testHandler{}

		router := NewRouter()
		router.Handle("GET", "/noop", WrapHttpHandler(h))

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/noop", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "" {
			t.Errorf("expected body %q, got %q", "", w.Body.String())
		}
	})
}
