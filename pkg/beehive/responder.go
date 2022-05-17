package beehive

import (
	"net/http"
)

// Responder is the required interface for beehive.Router to respond to a request. Returning a Responder instead of nil
// in a HandlerFunc chain, will stop chain execution and respond to the request. A Responder can be anything you want,
// including error responses, file responses, or custom responses. Utility Responder implementations are provided
// in the be-responder package.
type Responder interface {
	StatusCode(ctx *Context) int
	Body(ctx *Context) []byte
	Headers(ctx *Context)
	Cookies(ctx *Context) []*http.Cookie
}

// DefaultResponder is the default implementation of Responder. It returns a defined set of headers, the
// given status code in Status and the given body in Message. You can and should implement your own Responder
// and should even consider allocation global Responder variables where appropriate.
type DefaultResponder struct {
	Message []byte
	Status  int
}

// StatusCode returns the status code for the responder.
func (r *DefaultResponder) StatusCode(_ *Context) int {
	return r.Status
}

// Body returns the message body.
func (r *DefaultResponder) Body(_ *Context) []byte {
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
func (r *DefaultResponder) Headers(ctx *Context) {
	h := ctx.Request.Header

	h["Content-Type"] = defaultHeaderContentType
	h["X-Content-Type-Options"] = defaultHeaderContentTypeOptions
	h["X-Frame-Options"] = defaultHeaderFrameOptions
	h["X-XSS-Protection"] = defaultHeaderXssProtection
	h["Cache-Control"] = defaultHeaderCacheControl
}

// Cookies on the DefaultResponder returns no cookies.
func (r *DefaultResponder) Cookies(_ *Context) []*http.Cookie {
	return nil
}

var defaultPanicResponder = &DefaultResponder{
	Message: []byte("recovered from panic"),
	Status:  http.StatusInternalServerError,
}
