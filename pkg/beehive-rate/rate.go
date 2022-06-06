package beehive_rate

import (
	"strconv"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
)

type Limiter interface {
	Limit(key string) (int, time.Time)
}

func Limit(header string, limiter Limiter, limit int, responderFunc ResponderFunc) beehive.HandlerFunc {
	headerLimit := []string{strconv.Itoa(limit)}

	return func(ctx *beehive.Context) beehive.Responder {
		key := ctx.Request.Header.Get(header)

		w := ctx.ResponseWriter
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
