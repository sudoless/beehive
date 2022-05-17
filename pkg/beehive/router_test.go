package beehive

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestRouter_request_next(t *testing.T) {
	t.Parallel()

	handled := make([]string, 0)
	testHandler1 := func(ctx *Context) Responder {
		handled = append(handled, "1 pre")
		res := ctx.Next()
		handled = append(handled, "1 post")

		if finalRes := ctx.Next(); finalRes != nil {
			t.Errorf("expected nil, got %v", finalRes)
		}

		return res
	}
	testHandler2 := func(ctx *Context) Responder {
		handled = append(handled, "2 pre")
		if res := ctx.Next(); res != nil {
			return res
		}
		handled = append(handled, "2 post")

		return nil
	}
	testHandler3 := func(ctx *Context) Responder {
		handled = append(handled, "3 pre")
		res := ctx.Next()
		handled = append(handled, "3 post")

		return res
	}
	testHandler4 := func(ctx *Context) Responder {
		handled = append(handled, "4 do")

		if res := ctx.Next(); res != nil {
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
	router.HandleAny(methods, "/foo/bar", func(ctx *Context) Responder {
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
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
	t.Run("empty router", func(t *testing.T) {
		router := &Router{}
		router.Context = DefaultContext
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
			t.Logf("response_body=%q", w.Body.String())
		}
	})
}

func TestRouter_context(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		router := NewDefaultRouter()
		router.Context = func(r *http.Request) context.Context {
			return nil
		}

		ok := false
		router.Handle(http.MethodGet, "/foo/bar", func(ctx *Context) Responder {
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
		router.WhenContextDone = &DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusTeapot,
		}
		router.Context = func(_ *http.Request) context.Context {
			ctx, cc := context.WithCancel(context.Background())
			cc()
			return ctx
		}
		router.Handle("GET", "/foo/bar", func(ctx *Context) Responder {
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

	t.Run("default Recover", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatal("uncaught panic")
			}
		}()

		router := NewDefaultRouter()
		router.Handle("GET", "/foo/bar", func(_ *Context) Responder {
			panic("on purpose")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
		}

		if w.Body.String() != "recovered from panic" {
			t.Errorf("expected %q, got %q", "recovered from panic", w.Body.String())
		}
	})
	t.Run("defined Recover", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatal("uncaught panic")
			}
		}()

		router := NewDefaultRouter()
		router.Recover = func(ctx *Context, panicErr any) Responder {
			if panicErr != "on purpose" {
				t.Fatal("expected panicErr to be on purpose")
			}

			return &DefaultResponder{
				Message: nil,
				Status:  http.StatusTeapot,
			}
		}
		router.Handle("GET", "/foo/bar", func(_ *Context) Responder {
			panic("on purpose")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
		}
	})
	t.Run("defined Recover panic", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec != "double panic on purpose" {
					t.Fatal("expected panic to be double panic on purpose")
				}
			}
		}()

		router := NewDefaultRouter()
		router.Recover = func(ctx *Context, panicErr any) Responder {
			panic("double panic " + panicErr.(string))
		}
		router.Handle("GET", "/foo/bar", func(_ *Context) Responder {
			panic("on purpose")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		t.Fatal("should not reach here")
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

		testHandlerDummy := func(_ *Context) Responder {
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

func (c *cookieResponder) StatusCode(_ *Context) int {
	return http.StatusOK
}

func (c *cookieResponder) Body(_ *Context) []byte {
	return nil
}

func (c *cookieResponder) Headers(_ *Context) {}

func (c *cookieResponder) Cookies(_ *Context) []*http.Cookie {
	return c.cookies
}

func TestRouter_respond_cookies(t *testing.T) {
	t.Parallel()

	handler := func(ctx *Context) Responder {
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

	middleware := func(ctx *Context) Responder {
		w := ctx.ResponseWriter
		if w == nil {
			t.Fatalf("expected response writer, got nil")
		}

		h := w.Header()
		h.Set("X-Foo", "bar")
		h.Set("X-Bar", "foo")

		return nil
	}

	router := NewDefaultRouter()
	router.Handle("GET", "/foo", middleware, func(_ *Context) Responder {
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

func TestRouter_InServer_Shutdown(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var counter int32

	router := NewDefaultRouter()
	router.Handle("GET", "/sleep", func(ctx *Context) Responder {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&counter, 1)

		return &DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusAccepted,
		}
	})

	server := http.Server{
		Addr:    ":11406",
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				t.Errorf("expected %v, got %v", http.ErrServerClosed, err)
			}
		}
	}()

	time.Sleep(time.Millisecond * 10)

	client := &http.Client{}
	for iter := 0; iter < 100; iter++ {
		go func() {
			req, err := http.NewRequest(http.MethodGet, "http://:11406/sleep", nil)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			res, err := client.Do(req)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			_ = res.Body.Close()

			if res.StatusCode != http.StatusAccepted {
				t.Errorf("expected %d, got %d", http.StatusAccepted, res.StatusCode)
			}
		}()
	}

	time.Sleep(time.Millisecond * 10)

	if err := server.Shutdown(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	t.Log(counter)
	if counter != 100 {
		t.Errorf("expected %d, got %d", 100, counter)
	}
}

func TestRouter_Superfluous(t *testing.T) {
	t.Parallel()

	t.Run("new router", func(t *testing.T) {
		buffer := bytes.NewBuffer(nil)
		log.SetOutput(buffer)

		router := NewRouter()

		hijackingHandler := func(ctx *Context) Responder {
			w := ctx.ResponseWriter

			w.WriteHeader(http.StatusHTTPVersionNotSupported)
			_, _ = w.Write([]byte("hijacker"))

			return nil
		}

		router.Handle("GET", "/foo", hijackingHandler)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/foo", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusHTTPVersionNotSupported {
			t.Errorf("expected %d, got %d", http.StatusHTTPVersionNotSupported, w.Code)
		}
		if w.Body.String() != "hijacker" {
			t.Errorf("expected %s, got %s", "hijacker", w.Body.String())
		}

		t.Log(buffer.String())
	})
}
