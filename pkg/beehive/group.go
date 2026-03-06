package beehive

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
