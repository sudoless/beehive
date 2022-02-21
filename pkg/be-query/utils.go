package be_query

import "context"

type contextValuesKeyT struct{}

var contextValuesKey = &contextValuesKeyT{}

// ContextValues returns the *Values from the assigned context.Context.
func ContextValues(ctx context.Context) *Values {
	values, ok := ctx.Value(contextValuesKey).(*Values)
	if !ok {
		return nil
	}
	return values
}
