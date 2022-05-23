package beehive_responder

import (
	"github.com/sudoless/beehive/pkg/beehive"
)

// Status is a helper beehive.Responder that only implements StatusCode method and returns the given Code.
type Status struct {
	NoCookies
	NoBody

	Code int
}

func (s *Status) StatusCode(_ *beehive.Context) int {
	return s.Code
}
