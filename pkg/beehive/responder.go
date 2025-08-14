package beehive

import (
	"io"
	"net/http"
)

// Responder is the required interface for beehive.Router to respond to a request. Returning a Responder instead of nil
// in a HandlerFunc chain, will stop chain execution and respond to the request. A Responder can be anything you want,
// including error responses, file responses, or custom responses. Utility Responder implementations are provided
// in the be-responder package.
type Responder interface {
	// Respond is called by the Router when a request chain is complete by a returned Responder. If the returned
	// Responder is not nil, the Router will call Respond. Respond is in charge of writing to the http.ResponseWriter.
	// The Router.WhenNotFound, Router.WhenContextDone and Router.Recover functions are also called when any of the
	// aforementioned situations occur.
	Respond(ctx *Context)

	// StatusCode returns the status code of the Responder. This method must be called after Respond has been called.
	// Any call before Respond should not guarantee a valid status code. If the Responder does not implement StatusCode,
	// this must be documented and a default value must be returned.
	StatusCode(ctx *Context) int
}

// DefaultResponder is the default implementation of Responder. It returns a defined set of headers, the
// given status code in Status and the given body in Message. You can and should implement your own Responder
// and should even consider allocation global Responder variables where appropriate.
type DefaultResponder struct {
	Message string
	Status  int
}

// Respond satisfies the Responder interface and writes the given message and status code to the http.ResponseWriter.
func (r *DefaultResponder) Respond(ctx *Context) {
	w := ctx.ResponseWriter
	w.WriteHeader(r.Status)
	_, _ = io.WriteString(w, r.Message)
}

// StatusCode satisfies the Responder interface and returns the given status code. It should be safe to call before
// Respond has been called.
func (r *DefaultResponder) StatusCode(_ *Context) int {
	return r.Status
}

var (
	defaultPanicResponder = &DefaultResponder{
		Message: "recovered from panic",
		Status:  http.StatusInternalServerError,
	}

	defaultNotFoundResponder = &DefaultResponder{
		Message: "not found",
		Status:  http.StatusNotFound,
	}

	// TODO see what this is about.
	defaultContextDoneResponder = &DefaultResponder{ //nolint:unused
		Message: "context terminated",
		Status:  http.StatusGatewayTimeout,
	}
)
