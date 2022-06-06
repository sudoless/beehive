package beehive_responder

import (
	"go.sdls.io/beehive/pkg/beehive"
)

// Status is a helper beehive.Responder that only implements StatusCode method and returns the given Code.
type Status struct {
	NoBody

	Code int
}

func (s *Status) StatusCode(_ *beehive.Context) int {
	return s.Code
}
