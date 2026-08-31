package beehive_rate

import (
	"strconv"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
)

type Limiter interface {
	Limit(key string) (int, time.Time)
}

// Limit rejects requests once the Limiter reports the key at or above limit. It reports the state on every response
// through the X-RateLimit-Limit and X-RateLimit-Remaining headers, and on a rejection adds X-RateLimit-Reset and
// Retry-After, both in seconds until the limit resets.
func Limit(header string, limiter Limiter, limit int, responderFunc ResponderFunc) beehive.HandlerFunc {
	headerLimit := []string{strconv.Itoa(limit)}

	return func(ctx *beehive.Context) beehive.Responder {
		key := ctx.Request.Header.Get(header)

		h := ctx.ResponseWriter.Header()

		current, expiresAt := limiter.Limit(key)

		remaining := limit - current
		if remaining < 0 {
			remaining = 0
		}

		// Assigned directly to keep the handler allocation free, so the keys must already be in canonical form.
		h["X-Ratelimit-Limit"] = headerLimit
		h["X-Ratelimit-Remaining"] = []string{strconv.Itoa(remaining)}

		if current < limit {
			return nil
		}

		if !expiresAt.IsZero() {
			seconds := int(time.Until(expiresAt).Seconds())
			if seconds < 0 {
				seconds = 0
			}
			resetIn := []string{strconv.Itoa(seconds)}

			h["X-Ratelimit-Reset"] = resetIn
			h["Retry-After"] = resetIn
		}

		if responderFunc != nil {
			return responderFunc(key, limit, current, expiresAt)
		}

		return defaultResponder
	}
}
