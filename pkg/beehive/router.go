package beehive

import (
	"context"
	"net/http"
	"strconv"

	"github.com/sudoless/beehive/internal/node"
)

// Router is the core of the beehive package. It implements the Grouper interface for creating route groups or
// for applying middlewares. The Router has 5 Responder interfaces for certain core conditions which are documented
// on said interfaces.
type Router struct {
	// WhenNoResponse is called when all handlers in a chain are executed and a nil Responder is returned.
	WhenNoResponse Responder

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
	Recover func(ctx context.Context, r *http.Request, panicErr any) Responder

	methods map[string]*node.Trie
	group
}

// NewRouter returns an empty router with no implemented When... interfaces and the DefaultContext function.
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
	router.WhenNoResponse = &DefaultResponder{Status: http.StatusNoContent, Message: nil}

	return router
}

// DefaultContext returns the http.Request context. This is the same behaviour as returning a nil context.Context.
func DefaultContext(req *http.Request) context.Context {
	return req.Context()
}

// DebugRoutes is a helper function for debugging the router. It may be removed in the future.
func (router *Router) DebugRoutes() []string {
	routes := make([]string, 0)
	for method, trie := range router.methods {
		paths := trie.PathsHandlers()
		for path, data := range paths {
			handlers := data.Handlers.([]HandlerFunc)

			routes = append(routes, method+" "+path+" "+strconv.Itoa(len(handlers)))
		}
	}

	return routes
}

func (router *Router) respond(ctx context.Context, req *http.Request, w http.ResponseWriter, res Responder) {
	if res == nil {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte("beehive: router responder is nil"))
		return
	}

	body := res.Body(ctx, req)

	res.Headers(ctx, req, w.Header())

	for _, cookie := range res.Cookies(ctx, req) {
		http.SetCookie(w, cookie)
	}

	w.WriteHeader(res.StatusCode(ctx, req))

	if body != nil {
		_, _ = w.Write(body)
	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := router.Context(r)
	if ctx == nil {
		ctx = r.Context()
	}

	defer func() {
		if err := recover(); err != nil {
			if router.Recover != nil {
				router.respond(ctx, r, w, router.Recover(ctx, r, err))
			} else {
				router.respond(ctx, r, w, defaultPanicResponder)
			}
		}
	}()

	root := router.methods[r.Method]
	if root == nil {
		router.respond(ctx, r, w, router.WhenMethodNotAllowed)
		return
	}

	path := r.URL.Path
	n, err := root.Get(path)
	if err != nil {
		router.respond(ctx, r, w, router.WhenNotFound)
		return
	}

	handlers, ok := n.Data().([]HandlerFunc)
	if !ok || len(handlers) == 0 {
		router.respond(ctx, r, w, router.WhenNoResponse)
		return
	}

	extra := contextExtraPool.Get().(*contextExtra)
	*extra = contextExtra{
		w:           w,
		handlers:    handlers,
		handlersIdx: 0,
		ctx:         ctx,
		router:      router,
	}
	defer contextExtraPool.Put(extra)

	if res := router.next(extra, extra, r); res != nil {
		router.respond(ctx, r, w, res)
	} else {
		router.respond(ctx, r, w, router.WhenNoResponse)
	}
}

func (router *Router) next(ctx context.Context, extra *contextExtra, r *http.Request) Responder {
	for {
		select {
		case <-extra.ctx.Done():
			router.respond(ctx, r, extra.w, router.WhenContextDone)
			return nil
		default:
			if extra.handlersIdx >= len(extra.handlers) {
				return nil
			}

			res := extra.handlers[extra.handlersIdx](ctx, r)
			if res != nil {
				return res
			}

			extra.handlersIdx++
		}
	}
}
