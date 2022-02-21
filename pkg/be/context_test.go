package be

import (
	"context"
	"testing"
)

func Test_contextExtra_Context(t *testing.T) {
	t.Parallel()

	ctxKey := &struct{}{}

	ctx := &contextExtra{
		ctx: context.WithValue(context.Background(), ctxKey, "bar"),
	}

	deadline, _ := ctx.Deadline()
	if !deadline.IsZero() {
		t.Errorf("deadline should be zero")
	}

	if ctx.Done() != nil {
		t.Errorf("done should be nil")
	}

	if ctx.Err() != nil {
		t.Errorf("err should be nil")
	}

	out := ctx.Value(contextExtraKey)
	if out != ctx {
		t.Errorf("value should be ctx")
	}

	foo := ctx.Value(ctxKey)
	if foo.(string) != "bar" {
		t.Errorf("value should be bar")
	}
}

func Test_contextExtra_String(t *testing.T) {
	t.Parallel()

	ctxKey := &struct{}{}

	ctx := &contextExtra{
		ctx: context.WithValue(context.Background(), ctxKey, "bar"),
	}

	str := ctx.String()
	if str != "be.contextExtra(idx=0, handlers=[], ctx=(context.Background.WithValue(type *struct {}, val bar)))" {
		t.Errorf("bad context str, got '%s'", str)
	}
}
