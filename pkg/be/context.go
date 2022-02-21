package be

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type contextExtraKeyT struct{}

var contextExtraKey = &contextExtraKeyT{}

var contextExtraPool = &sync.Pool{
	New: func() interface{} {
		return &contextExtra{}
	},
}

type contextExtra struct {
	w           http.ResponseWriter
	ctx         context.Context
	router      *Router
	handlers    []HandlerFunc
	handlersIdx int
}

// String returns a formatted string with the contents of the context. This method has no guarantee of compatability
// between different versions of this package.
func (c *contextExtra) String() string {
	return fmt.Sprintf("be.contextExtra(idx=%d, handlers=%v, ctx=(%v))",
		c.handlersIdx, c.handlers, c.ctx)
}

func (c *contextExtra) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *contextExtra) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *contextExtra) Err() error {
	return c.ctx.Err()
}

func (c *contextExtra) Value(key interface{}) interface{} {
	if key == contextExtraKey {
		return c
	}

	return c.ctx.Value(key)
}
