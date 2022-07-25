package beehive_responder

import (
	"net/http"

	"go.sdls.io/beehive/pkg/beehive"
)

// NoBody implements beehive.Responder Body method by returning nil.
type NoBody struct{}

// test that NoBody implements beehive.Responder
var _ beehive.Responder = NoBody{}

func (n NoBody) Respond(_ *beehive.Context) {}

func (n NoBody) StatusCode(_ *beehive.Context) int { return http.StatusNoContent }
