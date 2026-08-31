package beehive

import "go.sdls.io/beehive/internal/trie"

// test that Router implements Grouper.
var _ Grouper = &Router{}

// Group creates a new routes group with the given prefix and the optional middleware which will be applied on all
// future calls to this group.
func (router *Router) Group(pathPrefix string, middleware ...HandlerFunc) Grouper {
	return &group{
		parent:     router,
		prefix:     groupPrefix(pathPrefix),
		middleware: middleware,
	}
}

// Handle registers a new request handlers to the given method and path.
func (router *Router) Handle(method, path string, handlers ...HandlerFunc) Grouper {
	if path == "" {
		panic("beehive: router path cannot be empty")
	}

	allHandlers := make([]HandlerFunc, len(router.middleware)+len(handlers))
	copy(allHandlers, router.middleware)
	copy(allHandlers[len(router.middleware):], handlers)

	if len(allHandlers) == 0 {
		panic("beehive: router handler is empty")
	}

	var radix *trie.Radix[[]HandlerFunc]
	for idx, m := range router.methods {
		if m.Name == method {
			radix = &router.methods[idx].radix
			break
		}
	}

	if radix == nil {
		router.methods = append(router.methods, methodGroup{
			Name:  method,
			radix: trie.Radix[[]HandlerFunc]{},
		})
		radix = &router.methods[len(router.methods)-1].radix
	}

	if !router.AllowRouteOverwrite && radix.Has(path) {
		panic("beehive: router route already defined")
	}

	radix.Add(path, allHandlers)

	return router
}

// HandleAny is a helper method for registering the same handlers on multiple methods for the same path.
func (router *Router) HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper {
	for _, method := range methods {
		router.Handle(method, path, handlers...)
	}

	return router
}

// With appends priority middleware (or handlers) to the Router. These middleware will be used first on any Handle.
// The middleware do not run on WhenNotFound or Recover.
func (router *Router) With(middleware ...HandlerFunc) Grouper {
	router.middleware = append(router.middleware, middleware...)
	return router
}
