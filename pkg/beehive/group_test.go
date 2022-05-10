package beehive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Group(t *testing.T) {
	t.Parallel()

	h := func(msg string) HandlerFunc {
		return func(_ context.Context, _ *http.Request) Responder {
			return &DefaultResponder{
				Message: []byte(msg),
				Status:  http.StatusOK,
			}
		}
	}

	m := func(_ context.Context, req *http.Request) Responder {
		if req.Header.Get("X-Test-Auth") != "yes" {
			return &DefaultResponder{
				Message: []byte("unauthorized"),
				Status:  http.StatusUnauthorized,
			}
		}

		return nil
	}

	t.Run("prefix", func(t *testing.T) {
		router := NewDefaultRouter()
		router.Group("/api").
			Handle("GET", "/health", h("/api/health")).
			Handle("GET", "/foo/bar", h("/api/foo/bar"))
		router.Group("/other").
			Handle("GET", "/health", h("/other/health"))

		paths := []string{
			"/api/health",
			"/api/foo/bar",
			"/other/health",
		}

		for _, path := range paths {
			r := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Body.String() != path {
				t.Errorf("expected %s, got %s", path, w.Body.String())
			}
		}
	})
	t.Run("middleware", func(t *testing.T) {
		router := NewDefaultRouter()
		api := router.Group("/api")
		{
			api.Handle("GET", "/health", h("/api/health"))
			apiAuth := api.Group("/auth", m)
			{
				apiAuth.Handle("GET", "/foo/bar", h("/api/auth/foo/bar"))
				apiAuth.Handle("GET", "/foo/bar/baz", h("/api/auth/foo/bar/baz"))
			}

			api.Handle("GET", "/foo/bar", h("/api/foo/bar"))
		}

		tests := []struct {
			path     string
			expected string
			withAuth bool
		}{
			{
				path:     "/api/health",
				expected: "/api/health",
				withAuth: false,
			},
			{
				path:     "/api/health",
				expected: "/api/health",
				withAuth: true,
			},
			{
				path:     "/api/foo/bar",
				expected: "/api/foo/bar",
				withAuth: false,
			},
			{
				path:     "/api/foo/bar",
				expected: "/api/foo/bar",
				withAuth: true,
			},
			{
				path:     "/api/auth/foo/bar",
				expected: "unauthorized",
				withAuth: false,
			},
			{
				path:     "/api/auth/foo/bar",
				expected: "/api/auth/foo/bar",
				withAuth: true,
			},
			{
				path:     "/api/auth/foo/bar/baz",
				expected: "unauthorized",
				withAuth: false,
			},
			{
				path:     "/api/auth/foo/bar/baz",
				expected: "/api/auth/foo/bar/baz",
				withAuth: true,
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%s(%t)", test.path, test.withAuth), func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", test.path, nil)
				if test.withAuth {
					r.Header.Set("X-Test-Auth", "yes")
				}
				router.ServeHTTP(w, r)

				if w.Body.String() != test.expected {
					t.Errorf("expected %s, got %s", test.expected, w.Body.String())
				}
			})
		}
	})
	t.Run("empty", func(t *testing.T) {
		counter := 0
		middleware := func(_ context.Context, _ *http.Request) Responder {
			counter++
			return nil
		}

		router := NewDefaultRouter()
		baseGroup := router.Group("", middleware)
		{
			baseGroup.Handle("GET", "/foo/bar", h("a"))
			baseGroup.Handle("GET", "/foo/bar/baz", h("b"))
			baseGroup.Handle("GET", "/bar/baz", h("c"))
		}

		paths := []string{
			"/foo/bar",
			"/foo/bar/baz",
			"/bar/baz",
		}

		for _, path := range paths {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", path, nil)

			router.ServeHTTP(w, r)
		}

		if counter != 3 {
			t.Errorf("expected counter to be 3, got %d", counter)
		}
	})
}

func TestRouter_Group_panic(t *testing.T) {
	t.Parallel()

	t.Run("wildcard ending", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		router := NewDefaultRouter()
		_ = router.Group("/api/*")
	})
}

func TestGroup_Handle_emptyPath(t *testing.T) {
	t.Parallel()

	router := NewDefaultRouter()
	g := router.Group("/has/prefix")
	g.Handle("GET", "", func(_ context.Context, _ *http.Request) Responder {
		return &DefaultResponder{
			Message: []byte("empty group path 1"),
			Status:  200,
		}
	})
	g.Handle("GET", "/", func(_ context.Context, _ *http.Request) Responder {
		return &DefaultResponder{
			Message: []byte("empty group path 2"),
			Status:  200,
		}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/has/prefix", nil)
	router.ServeHTTP(w, r)
	if w.Body.String() != "empty group path 1" {
		t.Errorf("expected %s, got %s", "empty group path 1", w.Body.String())
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/has/prefix/", nil)
	router.ServeHTTP(w, r)
	if w.Body.String() != "empty group path 2" {
		t.Errorf("expected %s, got %s", "empty group path 2", w.Body.String())
	}
}

func TestGroup_Handle_emptyPath_noPrefix(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	router := NewDefaultRouter()
	g := router.Group("")
	g.Handle("GET", "", func(_ context.Context, _ *http.Request) Responder {
		return &DefaultResponder{
			Message: []byte("empty group path 1"),
			Status:  200,
		}
	})

	t.Fatalf("expected panic")
}
