package be

import (
	"context"
	"net/http"
)

// Next can be used to pause execution in the handler chain, and continue with the next handler, then return to where
// Next was called. This is useful for things like authentication, logging, etc. where accessing the Responder is
// required.
func Next(ctx context.Context, r *http.Request) Responder {
	extra, ok := ctx.Value(contextExtraKey).(*contextExtra)
	if !ok {
		return nil
	}
	extra.handlersIdx++

	return extra.router.next(ctx, extra, r)
}

// ResponseWriter is a helper function that extracts and returns the http.ResponseWriter from contextExtra if it exists.
// This function is not recommended, and headers should be set using Responder.Headers method.
func ResponseWriter(ctx context.Context) http.ResponseWriter {
	extra, ok := ctx.Value(contextExtraKey).(*contextExtra)
	if !ok {
		return nil
	}

	return extra.w
}
