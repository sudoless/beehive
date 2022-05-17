package beehive_rate

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sudoless/beehive/pkg/beehive"
)

func Test_Responder(t *testing.T) {
	t.Parallel()

	router := beehive.NewDefaultRouter()
	router.Handle("GET", "/foo/bar", func(_ *beehive.Context) beehive.Responder {
		return defaultResponder
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/foo/bar", nil)

	router.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status code %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	if w.Body.String() != "" {
		t.Errorf("expected empty body, got %s", w.Body.String())
	}
}
