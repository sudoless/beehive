package beehive

// HandlerFunc is the function signature for both http Data and Middleware. The passed context.Context
// will be provided by the Router.Context function. The Responder can be nil if the HandlerFunc is used as
// a Middleware, but it can also return something to stop the execution of the next HandlerFunc.
// Request is a wrapper around the http.Request object, and more. Request also has a pointer to the current
// http.ResponseWriter (for use in Request.WriteHeaders) and pointers to the current HandlerFunc chain. The
// next HandlerFunc can be called by using Request.Next.
type HandlerFunc func(ctx *Context) Responder
