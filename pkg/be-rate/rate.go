package be_rate

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/sudoless/beehive/pkg/be"
)

type Limiter interface {
	Limit(key string) (int, time.Time)
}

func Limit(header string, limiter Limiter, limit int, responderFunc ResponderFunc) be.HandlerFunc {
	headerLimit := []string{strconv.Itoa(limit)}

	return func(ctx context.Context, req *http.Request) be.Responder {
		key := req.Header.Get(header)

		w := be.ResponseWriter(ctx)
		h := w.Header()

		current, expiresAt := limiter.Limit(key)

		h["X-RateLimit-Limit"] = headerLimit
		h["X-RateLimit-Remaining"] = []string{strconv.Itoa(limit - current)}

		if current >= limit {
			if !expiresAt.IsZero() {
				expiresAtSeconds := expiresAt.UTC().Second()

				h["X-RateLimit-Reset"] = []string{strconv.Itoa(expiresAtSeconds)}
				h["Retry-After"] = []string{strconv.Itoa(expiresAtSeconds - int(time.Now().UTC().Unix()))}
			}

			if responderFunc != nil {
				return responderFunc(key, limit, current, expiresAt)
			} else {
				return defaultResponder
			}
		}

		return nil
	}
}
