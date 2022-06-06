package beehive_responder

import (
	"net/http/httptest"
	"testing"

	"github.com/sudoless/beehive/pkg/beehive"
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

	if string(responder.Body(nil)) != want {
		t.Errorf("expected body '%s', got '%s'", want, string(responder.Body(nil)))
	}
}
