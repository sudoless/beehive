package beehive

import (
	"context"
	"net/http"

	"github.com/sudoless/beehive/internal/node"
)

// Router is the core of the beehive package. It implements the Grouper interface for creating route groups or
// for applying middlewares.
type Router struct {
	// WhenNotFound is called when the route does not match or the matched route has 0 handlers.
	WhenNotFound Responder

	// WhenMethodNotAllowed is called when the router has no routes defined for the requested method.
	WhenMethodNotAllowed Responder

	// WhenContextDone is called when the context is "done" (canceled or timed out).
	WhenContextDone Responder

	// Context is called to obtain a context for the request. By default, if a nil context.Context is returned then
	// the http.Request context is used.
	Context func(r *http.Request) context.Context

	// Recover is called when a panic occurs inside ServeHTTP.
	Recover func(ctx *Context, panicErr any) Responder

	methods map[string]*node.Trie
	group
}

// NewRouter returns an empty router with only the DefaultContext function.
func NewRouter() *Router {
	router := &Router{
		methods: make(map[string]*node.Trie),
		Context: DefaultContext,
	}

	router.group = group{
		router:     router,
		middleware: nil,
	}

	return router
}

// NewDefaultRouter returns an empty router with simple text/plain DefaultResponder implementation for the When...
// interfaces and the DefaultContext function.
func NewDefaultRouter() *Router {
	router := NewRouter()

	router.WhenNotFound = &DefaultResponder{Status: http.StatusNotFound, Message: []byte("not found")}
	router.WhenMethodNotAllowed = &DefaultResponder{Status: http.StatusMethodNotAllowed, Message: []byte("method not allowed")}
	router.WhenContextDone = &DefaultResponder{Status: http.StatusGatewayTimeout, Message: []byte("context finished, canceled or timed out")}

	return router
}

// DefaultContext returns the http.Request context. This is the same behaviour as returning a nil context.Context.
func DefaultContext(req *http.Request) context.Context {
	return req.Context()
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := router.Context(r)
	if c == nil {
		c = r.Context()
	}

	ctx := contextPool.Get().(*Context)
	*ctx = Context{
		ResponseWriter: w,
		Request:        r,
		handlersIdx:    0,
		Context:        c,
		router:         router,
	}
	router.serveHTTP(ctx)
	contextPool.Put(ctx)
}

func (router *Router) serveHTTP(ctx *Context) {
	r := ctx.Request

	defer func() {
		if err := recover(); err != nil {
			if router.Recover != nil {
				router.respond(ctx, router.Recover(ctx, err))
			} else {
				router.respond(ctx, defaultPanicResponder)
			}
		}
	}()

	root := router.methods[r.Method]
	if root == nil {
		router.respond(ctx, router.WhenMethodNotAllowed)
		return
	}

	path := r.URL.Path
	n, err := root.Get(path)
	if err != nil {
		router.respond(ctx, router.WhenNotFound)
		return
	}

	handlers, ok := n.Data().([]HandlerFunc)
	if !ok || len(handlers) == 0 {
		router.respond(ctx, router.WhenNotFound)
		return
	}
	ctx.handlers = handlers

	router.respond(ctx, router.next(ctx))
}

func (router *Router) respond(ctx *Context, res Responder) {
	if res == nil {
		return
	}

	w := ctx.ResponseWriter

	body := res.Body(ctx)

	for _, cookie := range res.Cookies(ctx) {
		http.SetCookie(w, cookie)
	}

	w.WriteHeader(res.StatusCode(ctx))

	if body != nil {
		_, _ = w.Write(body)
	}
}

func (router *Router) next(ctx *Context) Responder {
	for {
		select {
		case <-ctx.Context.Done():
			router.respond(ctx, router.WhenContextDone)
			return nil
		default:
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
}
