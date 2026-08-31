package beehive_rate

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
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
			Message: "ok",
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
				Message: "limited",
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

	for range 100 {
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

	for range 100 {
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
			Message: "ok",
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

// fixedLimiter reports a constant count and expiry, so header assertions do not depend on call order.
type fixedLimiter struct {
	current   int
	expiresAt time.Time
}

func (l fixedLimiter) Limit(string) (int, time.Time) {
	return l.current, l.expiresAt
}

func TestRateLimit_headersAreCanonical(t *testing.T) {
	t.Parallel()

	router := beehive.NewRouter()
	router.Handle(http.MethodGet, "/", Limit("X-Ip", fixedLimiter{current: 1}, 10, nil),
		func(_ *beehive.Context) beehive.Responder {
			return &beehive.DefaultResponder{Status: http.StatusNoContent}
		})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, r)

	if got := w.Header().Get("X-RateLimit-Limit"); got != "10" {
		t.Errorf("expected limit header %q, got %q", "10", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "9" {
		t.Errorf("expected remaining header %q, got %q", "9", got)
	}
}

func TestRateLimit_remainingIsClamped(t *testing.T) {
	t.Parallel()

	router := beehive.NewRouter()
	router.Handle(http.MethodGet, "/", Limit("X-Ip", fixedLimiter{current: 15}, 10, nil))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, r)

	if got := w.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("expected remaining header %q, got %q", "0", got)
	}
}

// Both X-RateLimit-Reset and Retry-After carry seconds until the limit resets, not a wall clock timestamp.
func TestRateLimit_resetIsDeltaSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expires  time.Duration
		wantLow  int
		wantHigh int
	}{
		{name: "in the future", expires: 90 * time.Second, wantLow: 88, wantHigh: 90},
		{name: "already expired", expires: -90 * time.Second, wantLow: 0, wantHigh: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			router := beehive.NewRouter()
			router.Handle(http.MethodGet, "/", Limit("X-Ip", fixedLimiter{
				current:   10,
				expiresAt: time.Now().Add(test.expires),
			}, 10, nil))

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			router.ServeHTTP(w, r)

			for _, header := range []string{"X-RateLimit-Reset", "Retry-After"} {
				seconds, err := strconv.Atoi(w.Header().Get(header))
				if err != nil {
					t.Errorf("expected %s to be an integer, got %q", header, w.Header().Get(header))
					continue
				}
				if seconds < test.wantLow || seconds > test.wantHigh {
					t.Errorf("expected %s between %d and %d, got %d", header, test.wantLow, test.wantHigh, seconds)
				}
			}
		})
	}
}
