package be

import (
	"context"
	"net/http"
)

// Responder is the required interface for be.Router to respond to a request. Returning a Responder instead of nil
// in a HandlerFunc chain, will stop chain execution and respond to the request. A Responder can be anything you want,
// including error responses, file responses, or custom responses. Utility Responder implementations are provided
// in the be-responder package.
type Responder interface {
	StatusCode(ctx context.Context, r *http.Request) int
	Body(ctx context.Context, r *http.Request) []byte
	Headers(ctx context.Context, r *http.Request, h http.Header)
	Cookies(ctx context.Context, r *http.Request) []*http.Cookie
}

// DefaultResponder is the default implementation of Responder. It returns a defined set of headers, the
// given status code in Status and the given body in Message. You can and should implement your own Responder
// and should even consider allocation global Responder variables where appropriate.
type DefaultResponder struct {
	Message []byte
	Status  int
}

// StatusCode returns the status code for the responder.
func (r *DefaultResponder) StatusCode(_ context.Context, _ *http.Request) int {
	return r.Status
}

// Body returns the message body.
func (r *DefaultResponder) Body(_ context.Context, _ *http.Request) []byte {
	return r.Message
}

var (
	defaultHeaderContentType        = []string{"text/plain; charset=utf-8"}
	defaultHeaderContentTypeOptions = []string{"nosniff"}
	defaultHeaderFrameOptions       = []string{"DENY"}
	defaultHeaderXssProtection      = []string{"1; mode=block"}
	defaultHeaderCacheControl       = []string{"no-cache, no-store, must-revalidate"}
)

// Headers on the DefaultResponder sets some fundamental, strict headers.
func (r *DefaultResponder) Headers(_ context.Context, _ *http.Request, h http.Header) {
	h["Content-Type"] = defaultHeaderContentType
	h["X-Content-Type-Options"] = defaultHeaderContentTypeOptions
	h["X-Frame-Options"] = defaultHeaderFrameOptions
	h["X-XSS-Protection"] = defaultHeaderXssProtection
	h["Cache-Control"] = defaultHeaderCacheControl
}

// Cookies on the DefaultResponder returns no cookies.
func (r *DefaultResponder) Cookies(_ context.Context, _ *http.Request) []*http.Cookie {
	return nil
}
