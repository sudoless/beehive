package beehive

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var contextPool = &sync.Pool{
	New: func() any {
		return &Context{}
	},
}

type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Context        context.Context //nolint:containedctx

	router      *Router
	handlers    []HandlerFunc
	handlersIdx int

	afters []func()
}

// String returns a formatted string with the contents of the context. This method has no guarantee of compatibility
// between different versions of this package.
func (c *Context) String() string {
	return fmt.Sprintf("beehive.Context(idx=%d, handlers=%v, ctx=(%v))",
		c.handlersIdx, c.handlers, c.Context)
}

// Deadline calls the underlying context.Context.Deadline() method.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.Context.Deadline()
}

// Done calls the underlying context.Context.Done() method.
func (c *Context) Done() <-chan struct{} {
	return c.Context.Done()
}

// Err calls the underlying context.Context.Err() method.
func (c *Context) Err() error {
	return c.Context.Err()
}

// Value calls the underlying context.Context.Value() method.
func (c *Context) Value(key any) any {
	return c.Context.Value(key)
}

// Next calls the next handler in the chain.
func (c *Context) Next() Responder {
	c.handlersIdx++
	return c.router.next(c)
}

// WithValue is a shortcut for Context.Context = context.WithValue(Context.Context, key, value) that also returns
// the Context such that it can be chained with other methods.
func (c *Context) WithValue(key, val any) *Context {
	c.Context = context.WithValue(c.Context, key, val)
	return c
}

// After appends a function to be called after router.serveHTTP is done, but before router.After is called.
func (c *Context) After(f func()) {
	c.afters = append(c.afters, f)
}
