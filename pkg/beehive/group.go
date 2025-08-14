package beehive

import "go.sdls.io/beehive/internal/trie"

// Grouper implements the abstraction layer for applying a handler or middleware on a group of routes.
type Grouper interface {
	// Group adds the given middleware to a *new* Grouper and returns it. The current Grouper is not modified.
	Group(pathPrefix string, middleware ...HandlerFunc) Grouper

	// Handle takes all the added middleware and the given handlers and registers them.
	Handle(method, path string, handlers ...HandlerFunc) Grouper

	// HandleAny takes all the added middleware and the given handlers and registers them on all given methods.
	HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper

	// With appends the given middlewar to the current Grouper and returns itself (for chaining). The changes here will
	// be "reflected" on Grouper relaying on parent chaining obtained by calling Group, With, Handle, etc. before or
	// after the call to With.
	With(middleware ...HandlerFunc) Grouper
}

type group struct {
	parent     Grouper
	prefix     string
	middleware []HandlerFunc
}

func (g *group) Group(pathPrefix string, middleware ...HandlerFunc) Grouper {
	return &group{
		parent:     g,
		prefix:     pathPrefix,
		middleware: middleware,
	}
}

func (g *group) Handle(method, path string, handlers ...HandlerFunc) Grouper {
	g.parent.Handle(method, g.prefix+path, append(g.middleware, handlers...)...)
	return g
}

func (g *group) HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper {
	g.parent.HandleAny(methods, g.prefix+path, append(g.middleware, handlers...)...)
	return g
}

func (g *group) With(middleware ...HandlerFunc) Grouper {
	g.middleware = append(g.middleware, middleware...)
	return g
}

// test that Router implements Grouper
var _ Grouper = &Router{}

// Group creates a new routes group with the given prefix and the optional middleware which will be applied on all
// future calls to this group.
func (router *Router) Group(pathPrefix string, middleware ...HandlerFunc) Grouper {
	if pathPrefix != "" && pathPrefix[len(pathPrefix)-1] == '*' {
		panic("beehive: router group path prefix cannot end with '*'")
	}

	return &group{
		parent:     router,
		prefix:     pathPrefix,
		middleware: middleware,
	}
}

// Handle registers a new request handlers to the given method and path.
func (router *Router) Handle(method, path string, handlers ...HandlerFunc) Grouper {
	if path == "" {
		panic("beehive: router path cannot be empty")
	}

	if len(handlers) == 0 {
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

	if !router.AllowRouteOverwrite {
		_, found := radix.Get(path)
		if found {
			panic("beehive: router route already defined")
		}
	}

	radix.Add(path, handlers)

	return router
}

// HandleAny is a helper method for registering the same handlers on multiple methods for the same path.
func (router *Router) HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper {
	for _, method := range methods {
		router.Handle(method, path, handlers...)
	}

	return router
}

func (router *Router) With(middleware ...HandlerFunc) Grouper {
	panic("beehive: router does not store middleware, use a Group/Grouper first")
}
