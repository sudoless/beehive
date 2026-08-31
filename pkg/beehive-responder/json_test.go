package beehive_responder

import (
	"net/http/httptest"
	"sync"
	"testing"

	"go.sdls.io/beehive/pkg/beehive"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	type Test struct {
		Name      string   `json:"name"`
		Age       int      `json:"age"`
		Roles     []string `json:"roles"`
		Empty     string   `json:"empty,omitempty"`
		NilString *string  `json:"nil_string"`
	}

	responder := &JSON{
		Object: &Test{
			Name:  "John Doe",
			Age:   30,
			Roles: []string{"admin", "user"},
		},
		Code: 200,
	}

	router := beehive.NewRouter()
	router.Handle("GET", "/", func(ctx *beehive.Context) beehive.Responder {
		return responder
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	if responder.data != nil {
		t.Error("data should be nil")
	}

	router.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Errorf("expected status code 200, got %d", w.Code)
	}

	want := `{"name":"John Doe","age":30,"roles":["admin","user"],"nil_string":null}`
	if w.Body.String() != want {
		t.Errorf("expected body '%s', got '%s'", want, w.Body.String())
	}

	if responder.data == nil {
		t.Error("data should not be nil")
	}
	if string(responder.data) != want {
		t.Errorf("expected data '%s', got '%s'", want, string(responder.data))
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	if string(responder.data) != want {
		t.Errorf("expected data '%s', got '%s'", want, string(responder.data))
	}

	w = httptest.NewRecorder()
	responder.Respond(&beehive.Context{
		ResponseWriter: w,
	})

	if w.Body.String() != want {
		t.Errorf("expected body '%s', got '%s'", want, w.Body.String())
	}

	if responder.StatusCode(nil) != w.Code {
		t.Errorf("expected status code %d, got %d", w.Code, responder.StatusCode(nil))
	}
}

// Content-Type must be set before WriteHeader, otherwise it never reaches the wire. Checking the recorder header map
// directly is not enough, it keeps accepting writes after the status has been committed.
func TestJSON_contentTypeReachesTheWire(t *testing.T) {
	t.Parallel()

	router := beehive.NewRouter()
	router.Handle("GET", "/", func(_ *beehive.Context) beehive.Responder {
		return &JSON{Object: map[string]bool{"ok": true}, Code: 200}
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	response := w.Result()
	defer func() { _ = response.Body.Close() }()

	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected content type 'application/json', got '%s'", got)
	}
}

// The package doc encourages allocating global Responder values, so the lazy data cache must be safe to share.
func TestJSON_sharedAcrossRequests(t *testing.T) { //nolint:paralleltest
	responder := &JSON{Object: map[string]bool{"ok": true}, Code: 200}

	router := beehive.NewRouter()
	router.Handle("GET", "/", func(_ *beehive.Context) beehive.Responder {
		return responder
	})

	const requests = 64

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(requests)

	for range requests {
		go func() {
			defer wg.Done()
			<-start
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
		}()
	}

	close(start)
	wg.Wait()
}
