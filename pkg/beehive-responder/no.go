package beehive_responder

import (
	"net/http"

	"github.com/sudoless/beehive/pkg/beehive"
)

// While nothing in here fully implements the beehive.Responder interface, they can be used in a struct to quickly
// fill and fully implement be.Responder.
// type ExampleResponder struct {
//     be_responder.NoCookies
//     be_responder.NoHeaders
// }

// NoCookies implements beehive.Responder Cookies method by returning nil.
type NoCookies struct{}

func (r *NoCookies) Cookies(_ *beehive.Context) []*http.Cookie {
	return nil
}

// NoBody implements beehive.Responder Body method by returning nil.
type NoBody struct{}

func (r *NoBody) Body(_ *beehive.Context) []byte {
	return nil
}

// NoHeaders implements beehive.Responder Headers method by setting no headers.
type NoHeaders struct{}

func (r *NoHeaders) Headers(_ *beehive.Context) {}
