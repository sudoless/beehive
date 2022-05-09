package beehive_responder

import (
	"context"
	"net/http"
)

// Status is a helper beehive.Responder that only implements StatusCode method and returns the given Code.
type Status struct {
	NoCookies
	NoHeaders
	NoBody

	Code int
}

func (s *Status) StatusCode(_ context.Context, _ *http.Request) int {
	return s.Code
}
