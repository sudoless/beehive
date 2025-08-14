package beehive

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
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
			Message: "solved",
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
		router.Context = func(_ *http.Request) context.Context {
			ctx, cc := context.WithCancel(context.Background())
			cc()
			return ctx
		}
		router.Handle("GET", "/foo/bar", func(ctx *Context) Responder {
			select {
			case <-ctx.Done():
				return &DefaultResponder{
					Message: "ok",
					Status:  http.StatusTeapot,
				}
			default:
				return nil
			}
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
				Message: "",
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
			Message: "ok",
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
	port := "11406"

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	var counter int32

	router := NewRouter()
	router.Handle("GET", "/sleep", func(ctx *Context) Responder {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&counter, 1)

		return &DefaultResponder{
			Message: "ok",
			Status:  http.StatusAccepted,
		}
	})

	server := http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	l, err := net.Listen("tcp4", "127.0.0.1:"+port)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(l.Addr())

	go func() {
		if serr := server.Serve(l); serr != nil {
			if !errors.Is(serr, http.ErrServerClosed) {
				t.Errorf("expected %v, got %v", http.ErrServerClosed, serr)
			}
		}
	}()

	client := &http.Client{}
	for iter := 0; iter < 100; iter++ {
		go func() {
			req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+port+"/sleep", nil)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			res, err := client.Do(req)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
				return
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

type noopResponder struct{}

func (n noopResponder) Respond(_ *Context) {}

func (n noopResponder) StatusCode(_ *Context) int {
	return 200
}

func BenchmarkRouter_ServeHTTP(b *testing.B) {
	responder := &noopResponder{}

	router := NewRouter()
	router.Context = func(r *http.Request) context.Context {
		return context.Background()
	}

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
	middleware2 := func(ctx *Context) Responder {
		select {
		case <-ctx.Done():
			return &DefaultResponder{
				Message: "context terminated",
				Status:  504,
			}
		default:
			trace = append(trace, "middleware2")
			return nil
		}
	}

	router.Handle("GET", "/foo", middleware1, middleware2, func(ctx *Context) Responder {
		trace = append(trace, "handler")
		return &DefaultResponder{
			Message: "ok",
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

type testResponderAfter struct {
	Value int
}

func (t testResponderAfter) Respond(ctx *Context) {
	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	_, _ = ctx.ResponseWriter.Write([]byte("ok"))
}

func (t testResponderAfter) StatusCode(_ *Context) int {
	return http.StatusOK
}

func TestRouter_After(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Context = func(r *http.Request) context.Context {
		return context.WithValue(context.Background(), "foo", "bar")
	}

	ran := false
	router.After = func(ctx *Context, res Responder) {
		ran = true

		if ctx.Value("foo").(string) != "bar" {
			t.Errorf("expected %s, got %s", "bar", ctx.Value("foo"))
		}

		if res == nil {
			t.Fatal("expected responder, got nil")
		}

		resV, ok := res.(testResponderAfter)
		if !ok {
			t.Fatalf("expected testResponderAfter, got %v", res)
		}

		if resV.Value != 123 {
			t.Errorf("expected %d, got %d", 123, resV.Value)
		}
	}

	router.Handle("GET", "/foo", func(_ *Context) Responder {
		return testResponderAfter{Value: 123}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)

	router.ServeHTTP(w, r)

	if !ran {
		t.Fatal("expected after to be ran")
	}
}

func TestRouter_After_panic(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Context = func(r *http.Request) context.Context {
		return context.WithValue(context.Background(), "foo", "bar")
	}

	ran := false
	router.After = func(ctx *Context, res Responder) {
		ran = true

		if ctx.Value("foo").(string) != "bar" {
			t.Errorf("expected %s, got %s", "bar", ctx.Value("foo"))
		}

		if res == nil {
			t.Fatal("expected responder, got nil")
		}

		if res.StatusCode(ctx) != 500 {
			t.Errorf("expected %d, got %d", 500, res.StatusCode(ctx))
		}
	}
	router.Recover = func(ctx *Context, err any) Responder {
		return &DefaultResponder{
			Message: "panic",
			Status:  500,
		}
	}

	router.Handle("GET", "/foo", func(_ *Context) Responder {
		panic(456)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)

	router.ServeHTTP(w, r)

	if !ran {
		t.Fatal("expected after to be ran")
	}
}

func testFuzzHandler(method, path string) HandlerFunc {
	return func(ctx *Context) Responder {
		return &DefaultResponder{
			Message: method + " " + path,
			Status:  200,
		}
	}
}

func FuzzRouter(f *testing.F) {
	method := "GET"
	paths := map[string]struct{}{
		"/foo/bar/baz":     {},
		"/foo/bar/buz":     {},
		"/foo/bar/bed":     {},
		"/foo/bar":         {},
		"/foo/bar/bug":     {},
		"/foo/biz/fiz":     {},
		"/hi":              {},
		"/contact":         {},
		"/co":              {},
		"/c":               {},
		"/a":               {},
		"/ab":              {},
		"/doc/":            {},
		"/doc/go_faq.html": {},
		"/doc/go1.html":    {},
		"/α":               {},
		"/β":               {},
	}

	router := NewRouter()
	for path := range paths {
		f.Add(path)

		router.Handle(method, path, testFuzzHandler(method, path))
	}

	for path := range paths {
		f.Add(path + "?foo=bar&baz=biz#anchor")
	}

	f.Fuzz(func(t *testing.T, path string) {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(method, path, nil)
			if err != nil {
				t.Skipf("bad request with path %q, %v", path, err)
				return
			}

			router.ServeHTTP(w, r)

			if _, ok := paths[r.URL.Path]; !ok {
				if w.Code != 404 {
					t.Fatalf("found not found path with status code %d", w.Code)
				}
			} else {
				if w.Code != 200 {
					t.Errorf("unexpected status code %d", w.Code)
				}

				if w.Body.String() != method+" "+r.URL.Path {
					t.Errorf("unexpected response body %q", w.Body.String())
				}
			}
		})
	})
}

func TestNewRouter_direct(t *testing.T) {
	router := &Router{}
	router.WhenNotFound = func(ctx *Context) Responder {
		return &DefaultResponder{
			Message: "not found",
			Status:  404,
		}
	}
	router.Recover = func(ctx *Context, panicErr any) Responder {
		t.Fatalf("unexpected panic: %v", panicErr)
		return nil
	}
	router.Context = func(r *http.Request) context.Context {
		return context.WithValue(context.Background(), "foo", "bar")
	}

	router.Handle("GET", "/foo", func(ctx *Context) Responder {
		return &DefaultResponder{
			Message: "foo",
			Status:  200,
		}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("unexpected status code %d", w.Code)
	}

	if w.Body.String() != "foo" {
		t.Fatalf("unexpected response body %q", w.Body.String())
	}
}
