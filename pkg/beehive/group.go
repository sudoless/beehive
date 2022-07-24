package beehive

// Grouper implements the abstraction layer for applying a handler or middleware on a group of routes.
type Grouper interface {
	Group(pathPrefix string, middleware ...HandlerFunc) Grouper
	Handle(method, path string, handlers ...HandlerFunc) Grouper
	HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper
}

type group struct {
	prefix     string
	router     *Router
	middleware []HandlerFunc
}

// Group creates a new routes group with the given prefix and the optional middleware which will be applied on all
// future calls to this group.
func (g *group) Group(pathPrefix string, middleware ...HandlerFunc) Grouper {
	if pathPrefix != "" && pathPrefix[len(pathPrefix)-1] == '*' {
		panic("beehive: router group path prefix cannot end with '*'")
	}

	return &group{
		prefix: g.prefix + pathPrefix,
		router: g.router,
		middleware: append(
			g.middleware,
			middleware...,
		),
	}
}

// Handle registers a new request handlers to the given method and path.
func (g *group) Handle(method, path string, handlers ...HandlerFunc) Grouper {
	if g.prefix == "" && path == "" {
		panic("beehive: router path cannot be empty")
	}

	if len(handlers) == 0 {
		panic("beehive: router handler is empty")
	}

	fullPath := method + g.prefix + path
	if !g.router.AllowRouteOverwrite {
		_, found := g.router.routes.Get(fullPath)
		if found {
			panic("beehive: router route already defined")
		}
	}

	g.router.routes.Add(fullPath, append(g.middleware, handlers...))

	return g
}

// HandleAny is a helper method for registering the same handlers on multiple methods for the same path.
func (g *group) HandleAny(methods []string, path string, handlers ...HandlerFunc) Grouper {
	for _, method := range methods {
		g.Handle(method, path, handlers...)
	}

	return g
}
