package beehive_query

import (
	"context"
	"testing"
)

func TestContextValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if v := ContextValues(ctx); v != nil {
		t.Fatalf("expected nil, got %v", v)
	}

	ctx = context.WithValue(ctx, contextValuesKey{}, &Values{
		dict:   nil,
		values: nil,
	})

	if v := ContextValues(ctx); v == nil {
		t.Fatalf("expected non-nil, got nil")
	}
}
