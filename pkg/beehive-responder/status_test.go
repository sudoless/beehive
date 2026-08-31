package beehive_responder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.sdls.io/beehive/pkg/beehive"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	responder := Status{Code: http.StatusNoContent}

	router := beehive.NewRouter()
	router.Handle("GET", "/", func(_ *beehive.Context) beehive.Responder {
		return responder
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Body.String() != "" {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}

	if got := responder.StatusCode(nil); got != w.Code {
		t.Errorf("expected StatusCode to report %d, got %d", w.Code, got)
	}
}
