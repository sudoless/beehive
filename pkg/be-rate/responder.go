package be_rate

import (
	"context"
	"net/http"
	"time"

	"github.com/sudoless/beehive/pkg/be"
	be_responder "github.com/sudoless/beehive/pkg/be-responder"
)

type ResponderFunc func(key string, limit, current int, expireAt time.Time) be.Responder

type Responder struct {
	be_responder.NoBody
	be_responder.NoHeaders
	be_responder.NoCookies
}

func (r *Responder) StatusCode(_ context.Context, _ *http.Request) int {
	return http.StatusTooManyRequests
}

var defaultResponder = &Responder{}
