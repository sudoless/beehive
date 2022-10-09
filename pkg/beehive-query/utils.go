package beehive_query

import "context"

type contextValuesKey struct{}

// ContextValues returns the *Values from the assigned context.Context.
func ContextValues(ctx context.Context) *Values {
	values, ok := ctx.Value(contextValuesKey{}).(*Values)
	if !ok {
		return nil
	}
	return values
}
