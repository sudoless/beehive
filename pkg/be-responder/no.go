package be_responder

import (
	"context"
	"net/http"
)

// While nothing in here fully implements the be.Responder interface, they can be used in a struct to quickly
// fill and fully implement be.Responder.
// type ExampleResponder struct {
//     be_responder.NoCookies
//     be_responder.NoHeaders
// }

// NoCookies implements be.Responder Cookies method by returning nil.
type NoCookies struct{}

func (r *NoCookies) Cookies(_ context.Context, _ *http.Request) []*http.Cookie {
	return nil
}

// NoBody implements be.Responder Body method by returning nil.
type NoBody struct{}

func (r *NoBody) Body(_ context.Context, _ *http.Request) []byte {
	return nil
}

// NoHeaders implements be.Responder Headers method by setting no headers.
type NoHeaders struct{}

func (r *NoHeaders) Headers(_ context.Context, _ *http.Request, _ http.Header) {}
