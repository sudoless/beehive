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
	"sync"
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
	r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
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
		t.Parallel()

		for idx, method := range methods {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(t.Context(), method, "/foo/bar", nil)
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
		t.Parallel()

		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar/baz", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("method not allowed", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), "HEAD", "/foo/bar/baz", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestRouter_default(t *testing.T) {
	t.Parallel()

	t.Run("no handlers", func(t *testing.T) {
		t.Parallel()

		router := NewRouter()
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
	t.Run("empty router", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, got nil")
			}
		}()

		router := &Router{}
		router.Context = func(r *http.Request) context.Context { return r.Context() }
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		t.FailNow()
	})
}

func TestRouter_context(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

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
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if !ok {
			t.Fatal("not ok")
		}
	})
	t.Run("closed", func(t *testing.T) {
		t.Parallel()

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
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
		}
	})
}

func TestRouter_recovery(t *testing.T) {
	t.Parallel()

	t.Run("default Recover", func(t *testing.T) {
		t.Parallel()

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
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected %d, got %d", http.StatusInternalServerError, w.Code)
		}

		if w.Body.String() != "recovered from panic" {
			t.Errorf("expected %q, got %q", "recovered from panic", w.Body.String())
		}
	})
	t.Run("defined Recover", func(t *testing.T) {
		t.Parallel()

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
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTeapot {
			t.Errorf("expected %d, got %d", http.StatusTeapot, w.Code)
		}
	})
	t.Run("defined Recover panic", func(t *testing.T) {
		t.Parallel()

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
		r := httptest.NewRequestWithContext(t.Context(), "GET", "/foo/bar", nil)
		router.ServeHTTP(w, r)

		t.Fatal("should not reach here")
	})
}

func TestRouter_Handle(t *testing.T) {
	t.Parallel()

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		router := NewRouter()
		router.Handle("GET", "")
	})
	t.Run("no handlers", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if rec := recover(); rec == nil {
				t.Error("expected panic")
			}
		}()

		router := NewRouter()
		router.Handle("GET", "/foo/bar")
	})
	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

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
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
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

func TestRouter_InServer_Shutdown(t *testing.T) { //nolint:paralleltest
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	const requests = 100

	var counter atomic.Int32

	// Shutdown must wait for in flight requests, so every request has to be inside the handler before it is called.
	var inHandler, done sync.WaitGroup
	inHandler.Add(requests)
	done.Add(requests)

	router := NewRouter()
	router.Handle("GET", "/sleep", func(_ *Context) Responder {
		inHandler.Done()
		time.Sleep(time.Millisecond * 100)
		counter.Add(1)

		return &DefaultResponder{
			Message: "ok",
			Status:  http.StatusAccepted,
		}
	})

	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := http.Server{Handler: router, ReadHeaderTimeout: time.Second}
	go func() {
		if serr := server.Serve(l); serr != nil && !errors.Is(serr, http.ErrServerClosed) {
			t.Errorf("expected %v, got %v", http.ErrServerClosed, serr)
		}
	}()

	url := "http://" + l.Addr().String() + "/sleep"
	client := &http.Client{Transport: &http.Transport{MaxConnsPerHost: requests}}

	for range requests {
		go func() {
			defer done.Done()

			req, rerr := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
			if rerr != nil {
				t.Errorf("expected no error, got %v", rerr)
				return
			}

			res, rerr := client.Do(req)
			if rerr != nil {
				t.Errorf("expected no error, got %v", rerr)
				return
			}
			_ = res.Body.Close()

			if res.StatusCode != http.StatusAccepted {
				t.Errorf("expected %d, got %d", http.StatusAccepted, res.StatusCode)
			}
		}()
	}

	inHandler.Wait()

	if err := server.Shutdown(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	done.Wait()

	if got := counter.Load(); got != requests {
		t.Errorf("expected %d, got %d", requests, got)
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
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
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
	cleanup := func() {}

	benchmarks := []struct {
		name    string
		handler HandlerFunc
	}{
		{
			name:    "plain",
			handler: func(_ *Context) Responder { return responder },
		},
		{
			// Reusing the afters backing array is what keeps this case off the allocator.
			name: "with after callback",
			handler: func(ctx *Context) Responder {
				ctx.After(cleanup)
				return responder
			},
		},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			router := NewRouter()
			router.Context = func(_ *http.Request) context.Context {
				return context.Background()
			}
			router.Handle("GET", "/foo/bar", benchmark.handler)

			r := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "/foo/bar", nil)
			w := noopResponseWriter{}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				router.ServeHTTP(w, r)
			}
		})
	}
}

// A wildcard match is not a route definition, so a concrete route may still be registered under a wildcard prefix.
// A concrete route and a wildcard sharing the exact same node is a collision the trie cannot represent, and panics.
func TestRouter_Handle_wildcardCollision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		first     string
		second    string
		wantPanic bool
	}{
		{name: "concrete then same path wildcard", first: "/foo", second: "/foo*", wantPanic: true},
		{name: "same path wildcard then concrete", first: "/foo*", second: "/foo", wantPanic: true},
		{name: "concrete then subtree wildcard", first: "/foo", second: "/foo/*"},
		{name: "subtree wildcard then concrete", first: "/foo/*", second: "/foo"},
		{name: "prefix wildcard then concrete below it", first: "/files/*", second: "/files/index.html"},
		{name: "concrete then prefix wildcard above it", first: "/files/index.html", second: "/files/*"},
		{name: "bare wildcard twice", first: "*", second: "*", wantPanic: true},
		{name: "bare wildcard then concrete", first: "*", second: "/foo"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := func(_ *Context) Responder { return nil }

			router := NewRouter()
			router.Handle("GET", test.first, handler)

			var panicValue any
			func() {
				defer func() { panicValue = recover() }()
				router.Handle("GET", test.second, handler)
			}()

			if test.wantPanic && panicValue == nil {
				t.Errorf("expected registering %q after %q to panic", test.second, test.first)
			}
			if !test.wantPanic && panicValue != nil {
				t.Errorf("expected registering %q after %q to succeed, panicked with %v",
					test.second, test.first, panicValue)
			}
		})
	}
}

func TestRouter_ServeHTTP_concreteRouteUnderWildcard(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Handle("GET", "/files/*", func(_ *Context) Responder {
		return &DefaultResponder{Message: "wildcard", Status: http.StatusOK}
	})
	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		router.Handle("GET", "/files/index.html", func(_ *Context) Responder {
			return &DefaultResponder{Message: "concrete", Status: http.StatusOK}
		})
	}()
	if panicValue != nil {
		t.Fatalf("expected registering /files/index.html under /files/* to succeed, panicked with %v", panicValue)
	}

	for path, want := range map[string]string{"/files/index.html": "concrete", "/files/other.txt": "wildcard"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		router.ServeHTTP(w, r)

		if got := w.Body.String(); got != want {
			t.Errorf("expected GET %s to respond %q, got %q", path, want, got)
		}
	}
}

func TestRouter_ServeHTTP_0alloc(t *testing.T) { //nolint:paralleltest
	responder := &noopResponder{}
	cleanup := func() {}

	tests := []struct {
		name    string
		handler HandlerFunc
	}{
		{
			name:    "plain",
			handler: func(_ *Context) Responder { return responder },
		},
		{
			// The pooled Context must keep the afters backing array across requests.
			name: "with after callback",
			handler: func(ctx *Context) Responder {
				ctx.After(cleanup)
				return responder
			},
		},
	}

	//nolint:paralleltest // AllocsPerRun is unreliable when subtests run concurrently
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter()
			router.Context = func(_ *http.Request) context.Context {
				return context.Background()
			}
			router.Handle("GET", "/foo/bar", test.handler)

			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo/bar", nil)
			w := noopResponseWriter{}

			allocs := testing.AllocsPerRun(100, func() {
				router.ServeHTTP(w, r)
			})
			if allocs != 0 {
				t.Errorf("expected 0 allocations, got %v", allocs)
			}
		})
	}
}

// Router.Context is a hook the Router installs by default, so a panic in it must reach Recover like any other
// panic that can still influence the response.
func TestRouter_ServeHTTP_contextFactoryPanic(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Context = func(_ *http.Request) context.Context {
		panic("context factory failed")
	}
	router.Recover = func(_ *Context, panicErr any) Responder {
		return &DefaultResponder{Message: panicErr.(string), Status: http.StatusInternalServerError}
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		router.ServeHTTP(w, r)
	}()

	if panicValue != nil {
		t.Fatalf("expected the panic to be recovered, it escaped with %v", panicValue)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
	if w.Body.String() != "context factory failed" {
		t.Errorf("expected Recover to see the panic value, got %q", w.Body.String())
	}
}

// A nil Context leaves the http.Request context in place, so a zero Router is usable.
func TestRouter_ServeHTTP_nilContextFactory(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Context = nil
	router.Handle("GET", "/", func(_ *Context) Responder {
		return &DefaultResponder{Message: "ok", Status: http.StatusOK}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		router.ServeHTTP(w, r)
	}()

	if panicValue != nil {
		t.Fatalf("expected a nil Context to be allowed, panicked with %v", panicValue)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body %q, got %q", "ok", w.Body.String())
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

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
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
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)

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
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)

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
			r, err := http.NewRequestWithContext(t.Context(), method, path, nil)
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

func TestRouter_With_basic(t *testing.T) {
	t.Parallel()

	trace := make([]string, 0)
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		trace = append(trace, "middleware")
		return nil
	})
	router.Handle("GET", "/foo", func(_ *Context) Responder {
		trace = append(trace, "handler")
		return &DefaultResponder{Status: http.StatusOK, Message: "ok"}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	expected := []string{"middleware", "handler"}
	if !reflect.DeepEqual(trace, expected) {
		t.Errorf("expected %v, got %v", expected, trace)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouter_With_order(t *testing.T) {
	t.Parallel()

	trace := make([]string, 0)
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		trace = append(trace, "mw1")
		return nil
	})
	router.With(func(_ *Context) Responder {
		trace = append(trace, "mw2")
		return nil
	})
	router.Handle("GET", "/foo", func(_ *Context) Responder {
		trace = append(trace, "handler")
		return &DefaultResponder{Status: http.StatusOK}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	expected := []string{"mw1", "mw2", "handler"}
	if !reflect.DeepEqual(trace, expected) {
		t.Errorf("expected %v, got %v", expected, trace)
	}
}

func TestRouter_With_short_circuit(t *testing.T) {
	t.Parallel()

	handlerCalled := false
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		return &DefaultResponder{Status: http.StatusUnauthorized, Message: "unauthorized"}
	})
	router.Handle("GET", "/foo", func(_ *Context) Responder {
		handlerCalled = true
		return &DefaultResponder{Status: http.StatusOK}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	if handlerCalled {
		t.Error("handler must not be called when router middleware short-circuits")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if w.Body.String() != "unauthorized" {
		t.Errorf("expected %q, got %q", "unauthorized", w.Body.String())
	}
}

func TestRouter_With_all_routes(t *testing.T) {
	t.Parallel()

	counter := 0
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		counter++
		return nil
	})

	router.HandleAny([]string{"GET", "POST", "PUT", "DELETE"}, "/foo", func(_ *Context) Responder {
		return &DefaultResponder{Status: http.StatusOK}
	})
	router.Handle("GET", "/bar", func(_ *Context) Responder {
		return &DefaultResponder{Status: http.StatusOK}
	})

	requests := []struct{ method, path string }{
		{"GET", "/foo"},
		{"POST", "/foo"},
		{"PUT", "/foo"},
		{"DELETE", "/foo"},
		{"GET", "/bar"},
	}
	for _, req := range requests {
		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(t.Context(), req.method, req.path, nil)
		router.ServeHTTP(w, r)
	}

	if counter != len(requests) {
		t.Errorf("expected middleware to run %d times, ran %d", len(requests), counter)
	}
}

func TestRouter_With_group(t *testing.T) {
	t.Parallel()

	trace := make([]string, 0)
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		trace = append(trace, "router-mw")
		return nil
	})
	router.Group("/api", func(_ *Context) Responder {
		trace = append(trace, "group-mw")
		return nil
	}).Handle("GET", "/foo", func(_ *Context) Responder {
		trace = append(trace, "handler")
		return &DefaultResponder{Status: http.StatusOK}
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/foo", nil)
	router.ServeHTTP(w, r)

	expected := []string{"router-mw", "group-mw", "handler"}
	if !reflect.DeepEqual(trace, expected) {
		t.Errorf("expected %v, got %v", expected, trace)
	}
}

func TestRouter_With_not_found(t *testing.T) {
	t.Parallel()

	mwCalled := false
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		mwCalled = true
		return nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/does-not-exist", nil)
	router.ServeHTTP(w, r)

	if mwCalled {
		t.Error("router middleware must not run for unmatched routes")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRouter_With_no_route_handlers(t *testing.T) {
	t.Parallel()

	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("unexpected panic: %v", rec)
		}
	}()

	called := false
	router := NewRouter()
	router.With(func(_ *Context) Responder {
		called = true
		return &DefaultResponder{Status: http.StatusOK, Message: "middleware-only"}
	})

	// No explicit handlers — must not panic because router.middleware is non-empty.
	router.Handle("GET", "/foo")

	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	if !called {
		t.Error("expected middleware to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouter_With_chaining(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	result := router.With(func(_ *Context) Responder { return nil })

	if result != router {
		t.Error("With must return the same *Router for chaining")
	}
}

func TestNewRouter_direct(t *testing.T) {
	t.Parallel()

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
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/foo", nil)
	router.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("unexpected status code %d", w.Code)
	}

	if w.Body.String() != "foo" {
		t.Fatalf("unexpected response body %q", w.Body.String())
	}
}

func TestRouter_ServeHTTP_nilRecover(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.Recover = nil

	afterRan := false
	router.After = func(_ *Context, _ Responder) { afterRan = true }
	router.Handle(http.MethodGet, "/boom", func(_ *Context) Responder { panic("boom") })

	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), "GET", "/boom", nil))
	}()

	if panicValue != "boom" {
		t.Errorf("expected the original panic to propagate, got %v", panicValue)
	}
	if !afterRan {
		t.Error("expected Router.After to run before the panic propagated")
	}
}
