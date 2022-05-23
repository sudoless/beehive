package beehive_rate

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sudoless/beehive/pkg/beehive"
)

type testRateLimiter struct {
	rates       map[string]int
	expires     map[string]time.Time
	expireAfter time.Duration
}

func (t *testRateLimiter) Limit(key string) (int, time.Time) {
	now := time.Now()

	rate := t.rates[key]
	rate++
	t.rates[key] = rate

	expireAt := t.expires[key]
	if expireAt.Before(now) {
		rate = 1
		t.rates[key] = rate

		expireAt = now.Add(t.expireAfter)
		t.expires[key] = expireAt
	}

	return rate, expireAt
}

func TestRateLimit(t *testing.T) {
	t.Parallel()

	counter := 0
	handler := func(ctx *beehive.Context) beehive.Responder {
		counter++

		return &beehive.DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusOK,
		}
	}

	testLimiter := &testRateLimiter{
		rates:       make(map[string]int),
		expires:     make(map[string]time.Time),
		expireAfter: time.Hour,
	}

	router := beehive.NewRouter()
	router.Handle("GET", "/foo/bar",
		Limit("X-Ip", testLimiter, 100, func(_ string, _, _ int, _ time.Time) beehive.Responder {
			return &beehive.DefaultResponder{
				Status:  http.StatusTooManyRequests,
				Message: []byte("limited"),
			}
		}),
		handler)
	router.Handle("GET", "/foo/bar/default",
		Limit("X-Ip", testLimiter, 0, nil),
		handler)

	for iter := 1; iter < 100; iter++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("X-Ip", "127.0.0.1")
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}
		if counter != iter {
			t.Errorf("expected %d, got %d", iter, counter)
		}
	}

	for iter := 0; iter < 100; iter++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar", nil)
		r.Header.Set("X-Ip", "127.0.0.1")
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status code %d, got %d", http.StatusTooManyRequests, w.Code)
		}
		if counter != 99 {
			t.Errorf("expected %d, got %d", 99, counter)
		}
		if w.Body.String() != "limited" {
			t.Errorf("expected %q, got %q", "limited", w.Body.String())
		}
	}

	for iter := 0; iter < 100; iter++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/foo/bar/default", nil)
		r.Header.Set("X-Ip", "127.0.0.1")
		router.ServeHTTP(w, r)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status code %d, got %d", http.StatusTooManyRequests, w.Code)
		}
		if counter != 99 {
			t.Errorf("expected %d, got %d", 99, counter)
		}
		if w.Body.String() != "" {
			t.Errorf("expected %q, got %q", "limited", w.Body.String())
		}
	}
}

func TestRateLimit_noKey(t *testing.T) {
	t.Parallel()

	counter := 0
	handler := func(ctx *beehive.Context) beehive.Responder {
		counter++

		return &beehive.DefaultResponder{
			Message: []byte("ok"),
			Status:  http.StatusOK,
		}
	}

	testLimiter := &testRateLimiter{
		rates:       make(map[string]int),
		expires:     make(map[string]time.Time),
		expireAfter: time.Hour,
	}

	router := beehive.NewRouter()
	router.Handle("GET", "/foo/bar",
		Limit("X-Ip", testLimiter, 100, nil),
		handler)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/foo/bar", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if testLimiter.rates[""] != 1 {
		t.Errorf("expected %d, got %d", 1, testLimiter.rates[""])
	}
}
