package beehive_responder

import (
	"go.sdls.io/beehive/pkg/beehive"
)

// Status is a helper beehive.Responder that only implements StatusCode method and returns the given Code.
type Status struct {
	Code int
}

func (s *Status) Respond(ctx *beehive.Context) {
	ctx.ResponseWriter.WriteHeader(s.Code)
}

// test that Status implements beehive.Responder
var _ beehive.Responder = &Status{}

func (s *Status) StatusCode(_ *beehive.Context) int {
	return s.Code
}
