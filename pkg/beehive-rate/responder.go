package beehive_rate

import (
	"net/http"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
	beehiveResponder "go.sdls.io/beehive/pkg/beehive-responder"
)

type ResponderFunc func(key string, limit, current int, expireAt time.Time) beehive.Responder

type Responder struct {
	beehiveResponder.NoBody
}

func (r *Responder) StatusCode(_ *beehive.Context) int {
	return http.StatusTooManyRequests
}

var defaultResponder = &Responder{}
