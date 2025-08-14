package beehive_rate

import (
	"net/http"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
)

type ResponderFunc func(key string, limit, current int, expireAt time.Time) beehive.Responder

type Responder struct{}

func (r *Responder) Respond(ctx *beehive.Context) {
	ctx.ResponseWriter.WriteHeader(http.StatusTooManyRequests)
}

// test that Responder implements beehive.Responder.
var _ beehive.Responder = &Responder{}

func (r *Responder) StatusCode(_ *beehive.Context) int {
	return http.StatusTooManyRequests
}

var defaultResponder = &Responder{}
