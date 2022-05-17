package beehive

import (
	"net/http"
)

// WrapHttpHandler wraps a standard library Go http.Handler with a Beehive HandlerFunc. The returned HandlerFunc will
// return a nil Responder, as the http.Handler will be responsible for writing the response.
func WrapHttpHandler(h http.Handler) HandlerFunc {
	return func(ctx *Context) Responder {
		h.ServeHTTP(ctx.ResponseWriter, ctx.Request)
		return nil
	}
}

// WrapHttpHandlerFunc wraps a standard library Go http.HandlerFunc with a Beehive HandlerFunc. The returned HandlerFunc
// will return a nil Responder, as the http.HandlerFunc will be responsible for writing the response.
func WrapHttpHandlerFunc(h http.HandlerFunc) HandlerFunc {
	return func(ctx *Context) Responder {
		h(ctx.ResponseWriter, ctx.Request)
		return nil
	}
}
