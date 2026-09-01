package beehive

import (
	"net/http"
)

// WrapHttpHandler wraps a standard library Go http.Handler with a Beehive HandlerFunc. The returned HandlerFunc will
// return a nil Responder, as the http.Handler will be responsible for writing the response. The wrapped handler sees
// the Context values, which costs the one allocation http.Request.WithContext makes.
func WrapHttpHandler(h http.Handler) HandlerFunc {
	return func(ctx *Context) Responder {
		h.ServeHTTP(ctx.ResponseWriter, requestWithContext(ctx))
		return nil
	}
}

// WrapHttpHandlerFunc wraps a standard library Go http.HandlerFunc with a Beehive HandlerFunc. The returned HandlerFunc
// will return a nil Responder, as the http.HandlerFunc will be responsible for writing the response. The wrapped
// handler sees the Context values, which costs the one allocation http.Request.WithContext makes.
func WrapHttpHandlerFunc(h http.HandlerFunc) HandlerFunc {
	return func(ctx *Context) Responder {
		h(ctx.ResponseWriter, requestWithContext(ctx))
		return nil
	}
}

// requestWithContext carries the Context values over to the wrapped handler. http.Request.WithContext panics on a nil
// context, so a handler that cleared Context.Context leaves the request as it came in.
func requestWithContext(ctx *Context) *http.Request {
	if ctx.Context == nil {
		return ctx.Request
	}

	return ctx.Request.WithContext(ctx.Context)
}
