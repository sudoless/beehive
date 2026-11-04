package beehive

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var chainPool = &sync.Pool{
	New: func() any {
		return &chain{}
	},
}

// Context is a request's view of the router. Each request gets its own Context, which is never reused, so it stays a
// valid context.Context for as long as anything holds it, including a goroutine that first runs after ServeHTTP
// returns.
type Context struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Context        context.Context //nolint:containedctx

	chain *chain
}

// chain is the routing state of a request in flight. It is pooled apart from the Context, and the Context lets go of
// it when ServeHTTP returns, so a Context outliving its request can never reach another request's chain.
type chain struct {
	router      *Router
	handlers    []HandlerFunc
	handlersIdx int

	afters []func()
}

// release drops what the request left on the chain, leaving it zero except for the afters backing array, which is
// kept so that registering a Context.After callback stays allocation free across pool reuse.
func (ch *chain) release() {
	clear(ch.afters)
	ch.afters = ch.afters[:0]

	ch.router = nil
	ch.handlers = nil
	ch.handlersIdx = 0
}

// String returns a formatted string with the contents of the context. This method has no guarantee of compatibility
// between different versions of this package.
func (c *Context) String() string {
	var ch chain
	if c.chain != nil {
		ch = *c.chain
	}

	return fmt.Sprintf("beehive.Context(idx=%d, handlers=%v, ctx=(%v))", ch.handlersIdx, ch.handlers, c.Context)
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

// Next calls the next handler in the chain. Outside a request, on a Context the router did not make or after
// ServeHTTP returned, there is no chain and Next returns nil.
func (c *Context) Next() Responder {
	if c.chain == nil {
		return nil
	}

	c.chain.handlersIdx++
	return c.chain.router.next(c)
}

// WithValue is a shortcut for Context.Context = context.WithValue(Context.Context, key, value) that also returns
// the Context such that it can be chained with other methods.
func (c *Context) WithValue(key, val any) *Context {
	c.Context = context.WithValue(c.Context, key, val)
	return c
}

// After appends a function to be called after router.serveHTTP is done, but before router.After is called. Outside a
// request the function is recorded and never called.
func (c *Context) After(f func()) {
	if c.chain == nil {
		c.chain = &chain{}
	}

	c.chain.afters = append(c.chain.afters, f)
}
