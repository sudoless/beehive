package beehive_rate

import (
	"context"
	"net/http"
	"time"

	"github.com/sudoless/beehive/pkg/beehive"
	be_responder "github.com/sudoless/beehive/pkg/beehive-responder"
)

type ResponderFunc func(key string, limit, current int, expireAt time.Time) beehive.Responder

type Responder struct {
	be_responder.NoBody
	be_responder.NoHeaders
	be_responder.NoCookies
}

func (r *Responder) StatusCode(_ context.Context, _ *http.Request) int {
	return http.StatusTooManyRequests
}

var defaultResponder = &Responder{}
