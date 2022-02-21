package be

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type testResponderNoPanic struct {
	t *testing.T
}

func (t testResponderNoPanic) Headers(_ context.Context, _ *http.Request, _ http.Header) {}

func (t testResponderNoPanic) StatusCode(_ context.Context, _ *http.Request) int {
	return 400
}

func (t testResponderNoPanic) Body(_ context.Context, req *http.Request) []byte {
	t.t.Errorf("unexpected panic: %v", req)
	return nil
}

func (t testResponderNoPanic) Cookies(_ context.Context, _ *http.Request) []*http.Cookie {
	return nil
}

func TestRouter_request_next(t *testing.T) {
	t.Parallel()

	handled := make([]string, 0)
	testHandler1 := func(ctx context.Context, req *http.Request) Responder {
		handled = append(handled, "1 pre")
		res := Next(ctx, req)
		handled = append(handled, "1 post")

		if finalRes := Next(ctx, req); finalRes != nil {
			t.Errorf("expected nil, got %v", finalRes)
		}

		return res
	}
	testHandler2 := func(ctx context.Context, req *http.Request) Responder {
		handled = append(handled, "2 pre")
		if res := Next(ctx, req); res != nil {
			return res
		}
		handled = append(handled, "2 post")

		return nil
	}
	testHandler3 := func(ctx context.Context, req *http.Request) Responder {
		handled = append(handled, "3 pre")
		res := Next(ctx, req)
		handled = append(handled, "3 post")

		return res
	}
	testHandler4 := func(ctx context.Context, req *http.Request) Responder {
		handled = append(handled, "4 do")

		if res := Next(ctx, req); res != nil {
			t.Errorf("expected nil, got %v", res)
		}

		return &DefaultResponder{
			Message: []byte("solved"),
			Status:  200,
		}
	}

	router := NewDefaultRouter()
	router.Handle("GET", "/foo/bar",
		testHandler1,
		testHandler2,
		testHandler3,
		testHandler4,
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/foo/bar", nil)
	router.ServeHTTP(w, r)

	expected := []string{
		"1 pre",
		"2 pre",
		"3 pre",
		"4 do",
		"3 post",
		"1 post",
	}

	if !reflect.DeepEqual(handled, expected) {
		t.Errorf("expected %v, got %v", expected, handled)
	}
}

func TestRouter_HandleAny(t *testing.T) {
	t.Parallel()

	methods := []string{"GET", "POST", "PUT"}
	router := NewDefaultRouter()

	counter := 0
	router.HandleAny(methods, "/foo/bar", func(ctx context.Context, req *http.Request) Responder {
		counter++
		return &DefaultResponder{
			Status: http.StatusOK,
		}
	})

	t.Run("methods", func(t *testing.T) {
		for idx, method := range methods {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(method, "/foo/bar", nil)
			router.ServeHTTP(w, r)

			if counter != idx+1 {
				t.Errorf("expected %d, got %d", idx+1, counter)
			}

			if w.Code != http.StatusOK {
				t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
			}
		}
	})
	t.Run("not found", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar/baz", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("method not allowed", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("HEAD", "/foo/bar/baz", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

func TestRouter_default(t *testing.T) {
	t.Parallel()

	t.Run("no handlers", func(t *testing.T) {
		router := NewDefaultRouter()
		router.WhenRecovering = testResponderNoPanic{t: t}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("empty router", func(t *testing.T) {
		router := &Router{}
		router.Context = DefaultContext
		router.WhenRecovering = testResponderNoPanic{t: t}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusGone {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestRouter_context(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		router := NewDefaultRouter()
		router.WhenRecovering = testResponderNoPanic{t: t}
		router.Context = func(r *http.Request) context.Context {
			return nil
		}

		ok := false
		router.Handle(http.MethodGet, "/foo/bar", func(ctx context.Context, _ *http.Request) Responder {
			if ctx == nil {
				t.Fatal("expected context, got nil")
			}

			ok = true
			return nil
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if !ok {
			t.Fatal("not ok")
		}
	})
	t.Run("closed", func(t *testing.T) {
		router := NewDefaultRouter()
		router.WhenRecovering = testResponderNoPanic{t: t}
		router.WhenContextDone = &DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusTeapot,
		}
		router.Context = func(_ *http.Request) context.Context {
			ctx, cc := context.WithCancel(context.Background())
			cc()
			return ctx
		}
		router.Handle("GET", "/foo/bar", func(ctx context.Context, request *http.Request) Responder {
			return nil
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
		}
	})
}

func TestRouter_recovery(t *testing.T) {
	t.Parallel()

	t.Run("no WhenRecovering", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatal("uncaught panic")
			}
		}()

		router := NewDefaultRouter()
		router.Handle("GET", "/foo/bar", func(_ context.Context, _ *http.Request) Responder {
			panic("on purpose")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusGone {
			t.Errorf("expected %d, got %d", http.StatusGone, w.Code)
		}
	})
	t.Run("WhenRecovering", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatal("uncaught panic")
			}
		}()

		router := NewDefaultRouter()
		router.WhenRecovering = &DefaultResponder{
			Message: nil,
			Status:  http.StatusTeapot,
		}
		router.Handle("GET", "/foo/bar", func(_ context.Context, _ *http.Request) Responder {
			panic("on purpose")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
		}
	})
}

func TestRouter_Handle(t *testing.T) {
	t.Parallel()

	t.Run("empty path", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		router := NewDefaultRouter()
		router.Handle("GET", "")
	})
	t.Run("no handlers", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		router := NewDefaultRouter()
		router.Handle("GET", "/foo/bar")
	})
	t.Run("duplicate", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		testHandlerDummy := func(_ context.Context, _ *http.Request) Responder {
			return nil
		}

		router := NewDefaultRouter()
		router.Handle("GET", "/foo/bar", testHandlerDummy)
		router.Handle("GET", "/foo/bar", testHandlerDummy)
	})
}

type cookieResponder struct {
	cookies []*http.Cookie
}

func (c *cookieResponder) StatusCode(_ context.Context, _ *http.Request) int {
	return http.StatusOK
}

func (c *cookieResponder) Body(_ context.Context, _ *http.Request) []byte {
	return nil
}

func (c *cookieResponder) Headers(_ context.Context, _ *http.Request, _ http.Header) {}

func (c *cookieResponder) Cookies(_ context.Context, _ *http.Request) []*http.Cookie {
	return c.cookies
}

func TestRouter_respond_cookies(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, r *http.Request) Responder {
		return &cookieResponder{
			cookies: []*http.Cookie{
				{
					Name:     "user",
					Value:    "foobar",
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   3600,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
			},
		}
	}

	router := NewDefaultRouter()
	router.Handle(http.MethodGet, "/foo", handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(w.Result().Cookies()))
	}

	cookie := cookies[0]
	if cookie.Name != "user" {
		t.Errorf("expected cookie name %s, got %s", "user", cookie.Name)
	}
	if cookie.Value != "foobar" {
		t.Errorf("expected cookie value %s, got %s", "foobar", cookie.Value)
	}
}

func Test_ResponseWriter(t *testing.T) {
	t.Parallel()

	middleware := func(ctx context.Context, _ *http.Request) Responder {
		w := ResponseWriter(ctx)
		if w == nil {
			t.Fatalf("expected response writer, got nil")
		}

		h := w.Header()
		h.Set("X-Foo", "bar")
		h.Set("X-Bar", "foo")

		return nil
	}

	router := NewDefaultRouter()
	router.Handle("GET", "/foo", middleware, func(_ context.Context, _ *http.Request) Responder {
		return &DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusOK,
		}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "ok" {
		t.Errorf("expected %s, got %s", "ok", w.Body.String())
	}

	h := w.Header()
	if h.Get("X-Foo") != "bar" {
		t.Errorf("expected %s, got %s", "bar", h.Get("X-Foo"))
	}
	if h.Get("X-Bar") != "foo" {
		t.Errorf("expected %s, got %s", "foo", h.Get("X-Bar"))
	}
}
