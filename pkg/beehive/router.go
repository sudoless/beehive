package beehive

import (
	"context"
	"net/http"

	"go.sdls.io/beehive/internal/trie"
)

type methodGroup struct {
	Name  string
	radix trie.Radix[[]HandlerFunc]
}

// Router is the core of the beehive package. It implements the Grouper interface for creating route groups or
// for applying middlewares.
type Router struct {
	// Context is called to obtain a context for the request. A nil Context, or one returning a nil context.Context,
	// leaves the http.Request context in place. A panic here reaches Recover like any other.
	Context func(r *http.Request) context.Context

	// WhenNotFound is called when the route does not match or the matched route has 0 handlers.
	WhenNotFound func(ctx *Context) Responder

	// Recover is called when a panic occurs while the response can still be influenced, which covers Context,
	// WhenNotFound, the handler chain and Responder.Respond. A panic inside Recover itself is not recovered.
	Recover func(ctx *Context, panicErr any) Responder

	// After is called after the request is handled and the response is sent. The *Context is still valid at this point.
	// The Responder is the response that was sent. If no response was sent, the Responder is nil. This method can be
	// used to do any cleanup without delaying the response.
	//
	// After runs before ServeHTTP returns, so slow cleanup holds the connection. A panic here, or in a Context.After
	// callback, is not recovered: the response is already written, so it propagates to net/http, which logs it and
	// closes the connection.
	After func(ctx *Context, res Responder)

	// AllowRouteOverwrite allows setting the same route multiple times. Not recommended.
	AllowRouteOverwrite bool

	methods    []methodGroup
	middleware []HandlerFunc
}

// NewRouter returns an empty router with only the DefaultContext function.
func NewRouter() *Router {
	router := &Router{
		Context: DefaultContext,
		Recover: func(ctx *Context, panicErr any) Responder {
			return defaultPanicResponder
		},
		WhenNotFound: func(ctx *Context) Responder {
			return defaultNotFoundResponder
		},
	}

	return router
}

// DefaultContext returns the http.Request context. This is the same behaviour as returning a nil context.Context.
func DefaultContext(req *http.Request) context.Context {
	return req.Context()
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := contextPool.Get().(*Context)
	*ctx = Context{
		ResponseWriter: w,
		Request:        r,
		Context:        r.Context(),
		router:         router,
		afters:         ctx.afters,
	}

	// Clearing the elements rather than the slice keeps the backing array across pool reuse, so registering a
	// Context.After callback stays allocation free, while still dropping the references those callbacks hold.
	defer func() {
		clear(ctx.afters)
		*ctx = Context{afters: ctx.afters[:0]}
		contextPool.Put(ctx)
	}()

	router.serveHTTP(ctx)
}

func (router *Router) serveHTTP(ctx *Context) {
	var res Responder
	r := ctx.Request

	defer func() {
		if err := recover(); err != nil {
			res = router.Recover(ctx, err)
			if res != nil {
				res.Respond(ctx)
			}
		}

		if len(ctx.afters) != 0 {
			for _, f := range ctx.afters {
				f()
			}
		}

		if router.After != nil {
			router.After(ctx, res)
		}
	}()

	if router.Context != nil {
		if c := router.Context(r); c != nil { //nolint:contextcheck
			ctx.Context = c
		}
	}

	var radix *trie.Radix[[]HandlerFunc]
	for idx, method := range router.methods {
		if method.Name == r.Method {
			radix = &router.methods[idx].radix
			break
		}
	}

	if radix == nil {
		if res = router.WhenNotFound(ctx); res != nil {
			res.Respond(ctx)
		}
		return
	}

	data, found := radix.Get(r.URL.Path)
	if !found {
		if res = router.WhenNotFound(ctx); res != nil {
			res.Respond(ctx)
		}
		return
	}

	ctx.handlers = data
	if len(ctx.handlers) == 0 {
		if res = router.WhenNotFound(ctx); res != nil {
			res.Respond(ctx)
		}
		return
	}

	res = router.next(ctx)
	if res != nil {
		res.Respond(ctx)
	}
}

func (router *Router) next(ctx *Context) Responder {
	for {
		if ctx.handlersIdx >= len(ctx.handlers) {
			return nil
		}

		res := ctx.handlers[ctx.handlersIdx](ctx)
		if res != nil {
			return res
		}

		ctx.handlersIdx++
	}
}
