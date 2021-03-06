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

	router := NewRouter()
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
	router := NewRouter()

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

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestRouter_default(t *testing.T) {
	t.Parallel()

	t.Run("no handlers", func(t *testing.T) {
		router := NewRouter()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("empty router", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()

		router := &Router{}
		router.Context = DefaultContext
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		t.FailNow()
	})
}

func TestRouter_context(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		router := NewRouter()
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
		router := NewRouter()
		router.WhenContextDone = func(_ *Context) Responder {
			return &DefaultResponder{
				Message: []byte("ok"),
				Status:  http.StatusTeapot,
			}
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

		router := NewRouter()
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

		router := NewRouter()
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

		router := NewRouter()
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

		router := NewRouter()
		router.Handle("GET", "")
	})
	t.Run("no handlers", func(t *testing.T) {
		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		router := NewRouter()
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

		router := NewRouter()
		router.Handle("GET", "/foo/bar", testHandlerDummy)
		router.Handle("GET", "/foo/bar", testHandlerDummy)
	})
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

	router := NewRouter()
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
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var counter int32

	router := NewRouter()
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

type noopResponseWriter struct{}

func (n noopResponseWriter) Header() http.Header { return http.Header{} }

func (n noopResponseWriter) Write(i []byte) (int, error) {
	return len(i), nil
}

func (n noopResponseWriter) WriteHeader(_ int) {}

func BenchmarkRouter_ServeHTTP(b *testing.B) {
	responder := &DefaultResponder{
		Message: []byte("ok"),
		Status:  200,
	}

	router := NewRouter()
	router.Handle("GET", "/foo/bar", func(ctx *Context) Responder {
		return responder
	})

	r := httptest.NewRequest(http.MethodGet, "/foo/bar", nil)
	w := noopResponseWriter{}

	b.ReportAllocs()
	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		router.ServeHTTP(w, r)
	}
}

func TestRouter_ServeHTTP_contextDone(t *testing.T) {
	t.Parallel()

	tCtx, cc := context.WithCancel(context.Background())
	defer cc()

	router := NewRouter()
	router.Context = func(_ *http.Request) context.Context {
		return tCtx
	}

	trace := make([]string, 0)
	middleware1 := func(ctx *Context) Responder {
		cc()

		trace = append(trace, "middleware1 start")
		res := ctx.Next()
		trace = append(trace, "middleware1 end")

		return res
	}
	middleware2 := func(_ *Context) Responder {
		trace = append(trace, "middleware2")
		return nil
	}

	router.Handle("GET", "/foo", middleware1, middleware2, func(ctx *Context) Responder {
		trace = append(trace, "handler")
		return &DefaultResponder{
			Message: []byte("ok"),
			Status:  200,
		}
	})

	r := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if !reflect.DeepEqual(trace, []string{
		"middleware1 start",
		"middleware1 end",
	}) {
		t.Errorf("expected %v, got %v", []string{
			"middleware1 start",
			"middleware1 end",
		}, trace)
	}

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected %d, got %d", http.StatusGatewayTimeout, w.Code)
	}
	if w.Body.String() != "context terminated" {
		t.Errorf("expected %s, got %s", "context terminated", w.Body.String())
	}
}

type testResponderOrder struct {
	statusCode int
	body       int
	callback   func(string)
}

func (t *testResponderOrder) StatusCode(_ *Context) int {
	t.callback("statusCode")
	t.statusCode++
	return 200
}

func (t *testResponderOrder) Body(_ *Context) []byte {
	t.callback("body")
	t.body++
	return []byte("ok")
}

func TestRouter_ServeHTTP_ResponderOrder(t *testing.T) {
	t.Parallel()

	count := 0
	responder := &testResponderOrder{
		statusCode: 0,
		body:       0,
		callback: func(s string) {
			if s == "body" && count != 0 {
				t.Errorf("expected 'body' callback at count %d, got %d", 0, count)
			}
			if s == "statusCode" && count != 1 {
				t.Errorf("expected 'statusCode' callback at count %d, got %d", 1, count)
			}
			count++
		},
	}

	router := NewRouter()
	router.Handle("GET", "/foo", func(ctx *Context) Responder {
		return responder
	})

	r := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if responder.statusCode != 1 {
		t.Errorf("expected %d, got %d", 1, responder.statusCode)
	}

	if responder.body != 1 {
		t.Errorf("expected %d, got %d", 1, responder.body)
	}
}
