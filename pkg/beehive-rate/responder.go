package beehive_rate

import (
	"net/http"
	"time"

	"github.com/sudoless/beehive/pkg/beehive"
	beehiveResponder "github.com/sudoless/beehive/pkg/beehive-responder"
)

type ResponderFunc func(key string, limit, current int, expireAt time.Time) beehive.Responder

type Responder struct {
	beehiveResponder.NoBody
}

func (r *Responder) StatusCode(_ *beehive.Context) int {
	return http.StatusTooManyRequests
}

var defaultResponder = &Responder{}
