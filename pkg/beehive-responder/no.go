package beehive_responder

import (
	"go.sdls.io/beehive/pkg/beehive"
)

// NoBody implements beehive.Responder Body method by returning nil.
type NoBody struct{}

func (r *NoBody) Body(_ *beehive.Context) []byte {
	return nil
}
